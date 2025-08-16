package gnb

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/free5gc/aper"
)

const (
	IS_NEXT_EXTENSION_HEADER = 0x04
	IS_SEQUENCE_NUMBER       = 0x02
	IS_N_PDU_NUMBER          = 0x01

	NEXT_EXTENSION_HEADER_TYPE_NO_MORE_EXTENSION_HEADERS = 0x00

	NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER        = 0x85
	NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER_LENGTH = 2
)

type TeidGenerator struct {
	teids sync.Map
	mtx   sync.Mutex
}

func NewTeidGenerator() *TeidGenerator {
	return &TeidGenerator{
		teids: sync.Map{},
		mtx:   sync.Mutex{},
	}
}

func (t *TeidGenerator) AllocateTeid() aper.OctetString {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	for i := 1; i <= 65535; i++ {
		if _, exists := t.teids.Load(int64(i)); !exists {
			t.teids.Store(int64(i), true)

			teid, err := hex.DecodeString(t.formatAsString(int64(i)))
			if err != nil {
				panic(fmt.Errorf("error decode teid: %v", err))
			}

			return aper.OctetString(teid)
		}
	}

	return aper.OctetString{}
}

func (t *TeidGenerator) ReleaseTeid(teid aper.OctetString) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	value := t.deFormatFromString(hex.EncodeToString(teid))

	if _, exists := t.teids.Load(value); exists {
		t.teids.Delete(value)
	} else {
		panic(fmt.Errorf("attempting to release teid %s that is not allocated", hex.EncodeToString(teid)))
	}
}

func (t *TeidGenerator) formatAsString(teid int64) string {
	return fmt.Sprintf("%08x", teid)
}

func (t *TeidGenerator) deFormatFromString(teid string) int64 {
	teidInt, err := strconv.ParseInt(teid, 16, 64)
	if err != nil {
		panic(fmt.Errorf("error deformat teid: %v", err))
	}
	return teidInt
}

// get packet with GTP header from gtpChannel and forward to N3 connection
func forwardGtpPacketToN3Conn(ctx context.Context, n3Conn *net.UDPConn, gtpChannel chan []byte, gnbLogger *logger.GnbLogger) {
	for {
		select {
		case <-ctx.Done():
			gnbLogger.GtpLog.Debugln("Forward GTP packet to N3 connection stopped")
			return
		case packet := <-gtpChannel:
			n, err := n3Conn.Write(packet)
			if err != nil {
				gnbLogger.GtpLog.Errorf("Error writing GTP packet to N3 connection: %v", err)
				return
			}
			gnbLogger.GtpLog.Tracef("Forwarded %d bytes of GTP packet to N3 connection", n)
			gnbLogger.GtpLog.Debugln("Forwarded GTP packet to N3 connection")
		}
	}
}

// receive GTP packet from N3 connection and forward to UE according to the GTP header's TEID
func receiveGtpPacketFromN3Conn(ctx context.Context, n3Conn *net.UDPConn, gnbLogger *logger.GnbLogger, teidToConn *sync.Map) {
	buffer := make([]byte, 4096)
	for {
		n, err := n3Conn.Read(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			gnbLogger.GtpLog.Warnf("Error reading GTP packet from N3 connection: %v", err)
		}
		gnbLogger.GtpLog.Tracef("Received %d bytes of GTP packet from N3 connection: %+v", n, buffer[:n])
		gnbLogger.GtpLog.Tracef("Received %d bytes of GTP packet from N3 connection", n)
		go forwardPacketToUe(buffer[:n], teidToConn, gnbLogger)
	}
}

// format GTP packet and write to gtpChannel
func formatGtpPacketAndWriteToGtpChannel(teid aper.OctetString, packet []byte, gtpChannel chan []byte, gnbLogger *logger.GnbLogger) {
	gtpHeader := make([]byte, 12)

	gtpHeader[0] = 0x32
	gtpHeader[1] = 0xff
	binary.BigEndian.PutUint16(gtpHeader[2:], uint16(len(packet)+4))
	copy(gtpHeader[4:], teid)
	gtpHeader[8], gtpHeader[9], gtpHeader[10], gtpHeader[11] = 0x00, 0x00, 0x00, 0x00

	gtpPacket := append(gtpHeader, packet...)
	gnbLogger.GtpLog.Tracef("Formatted GTP packet: %+v", gtpPacket)

	gtpChannel <- gtpPacket
	gnbLogger.GtpLog.Tracef("Wrote %d bytes of GTP packet to gtpChannel", len(gtpPacket))
	gnbLogger.GtpLog.Debugln("Wrote GTP packet to gtpChannel")
}

// forward packet to UE according to the GTP header's TEID
func forwardPacketToUe(gtpPacket []byte, teidToConn *sync.Map, gnbLogger *logger.GnbLogger) {
	teid, payload, err := parseGtpPacket(gtpPacket)
	if err != nil {
		gnbLogger.GtpLog.Warnf("Error parsing GTP packet: %v", err)
		return
	}
	gnbLogger.GtpLog.Tracef("Parsed GTP packet: TEID: %s, Payload: %+v", teid, payload)

	conn, found := teidToConn.Load(teid)
	if !found {
		gnbLogger.GtpLog.Warnf("No connection found for TEID: %s", teid)
		return
	}
	gnbLogger.GtpLog.Debugf("Loaded connection %s for TEID: %s", conn.(net.Conn).RemoteAddr(), teid)

	n, err := conn.(net.Conn).Write(payload)
	if err != nil {
		gnbLogger.GtpLog.Warnf("Error writing GTP packet to UE: %v", err)
		return
	}
	gnbLogger.GtpLog.Tracef("Forwarded %d bytes of GTP packet to UE", n)
	gnbLogger.GtpLog.Debugln("Forwarded GTP packet to UE")
}

// parse GTP packet, will return the TEID and payload
func parseGtpPacket(gtpPacket []byte) (string, []byte, error) {
	basicHeader, headerLength := gtpPacket[:8], 8

	isNextExtensionHeader, isSequenceNumber, isNPDUNumber := false, false, false

	if basicHeader[0]&IS_NEXT_EXTENSION_HEADER != 0 {
		isNextExtensionHeader = true
	}

	if basicHeader[0]&IS_SEQUENCE_NUMBER != 0 {
		isSequenceNumber = true
	}

	if basicHeader[0]&IS_N_PDU_NUMBER != 0 {
		isNPDUNumber = true
	}

	if isNextExtensionHeader || isSequenceNumber || isNPDUNumber {
		headerLength += 3
	}

	if !isNextExtensionHeader {
		return hex.EncodeToString(basicHeader[4:]), gtpPacket[headerLength:], nil
	}

	for {
		switch gtpPacket[headerLength] {
		case NEXT_EXTENSION_HEADER_TYPE_NO_MORE_EXTENSION_HEADERS:
			headerLength += 1
			return hex.EncodeToString(basicHeader[4:]), gtpPacket[headerLength:], nil
		case NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER:
			extensionHeaderLength := gtpPacket[headerLength+1]
			headerLength += 2 + int(extensionHeaderLength)*NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER_LENGTH
		default:
			return "", nil, fmt.Errorf("unknown GTP extension header type: %d", gtpPacket[headerLength])
		}
	}
}
