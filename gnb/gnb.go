package gnb

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	consoleModel "github.com/Alonza0314/free-ran-ue/console/model"
	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/util"
	"github.com/free5gc/aper"
	"github.com/free5gc/nas"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/sctp"
	"github.com/gin-gonic/gin"
)

type xnInterface struct {
	xnIp   string
	xnPort int
}

type api struct {
	ip   string
	port int

	router *gin.Engine
	server *http.Server
}

type Gnb struct {
	amfN2Ip string
	ranN2Ip string
	upfN3Ip string
	ranN3Ip string

	ranControlPlaneIp string
	ranDataPlaneIp    string

	xnIp string

	amfN2Port int
	ranN2Port int
	upfN3Port int
	ranN3Port int

	ranControlPlanePort int
	ranDataPlanePort    int

	xnPort int

	n2Conn *sctp.SCTPConn
	n3Conn *net.UDPConn

	ngapPpid uint32

	gnbId   []byte
	gnbName string

	plmnId ngapType.PLMNIdentity
	tai    ngapType.TAI
	snssai ngapType.SNSSAI

	staticNrdc bool

	xnInterface

	ranControlPlaneListener *net.Listener
	ranDataPlaneListener    *net.Listener
	xnListener              *net.Listener

	ranUeConns sync.Map
	xnUeConns  sync.Map
	teidToConn sync.Map

	gtpChannel chan []byte

	ranUeNgapIdGenerator *RanUeNgapIdGenerator
	teidGenerator        *TeidGenerator

	api

	*logger.GnbLogger
}

func NewGnb(config *model.GnbConfig, gnbLogger *logger.GnbLogger) *Gnb {
	gnbId, err := hex.DecodeString(config.Gnb.GnbId)
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error decoding gnbId to bytes: %v", err)
		return nil
	}

	plmnId, err := util.PlmnIdToNgap(models.PlmnId{
		Mcc: config.Gnb.PlmnId.Mcc,
		Mnc: config.Gnb.PlmnId.Mnc,
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting plmnId to ngap: %v", err)
		return nil
	}

	tai, err := util.TaiToNgap(models.Tai{
		Tac: config.Gnb.Tai.Tac,
		PlmnId: &models.PlmnId{
			Mcc: config.Gnb.Tai.BroadcastPlmnId.Mcc,
			Mnc: config.Gnb.Tai.BroadcastPlmnId.Mnc,
		},
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting tai to ngap: %v", err)
		return nil
	}

	sstInt, err := strconv.Atoi(config.Gnb.Snssai.Sst)
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting sst to int: %v", err)
		return nil
	}
	snssai, err := util.SNssaiToNgap(models.Snssai{
		Sst: int32(sstInt),
		Sd:  config.Gnb.Snssai.Sd,
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting snssai to ngap: %v", err)
		return nil
	}

	return &Gnb{
		amfN2Ip:           config.Gnb.AmfN2Ip,
		ranN2Ip:           config.Gnb.RanN2Ip,
		upfN3Ip:           config.Gnb.UpfN3Ip,
		ranN3Ip:           config.Gnb.RanN3Ip,
		ranControlPlaneIp: config.Gnb.RanControlPlaneIp,
		ranDataPlaneIp:    config.Gnb.RanDataPlaneIp,
		xnIp:              config.Gnb.XnIp,

		amfN2Port:           config.Gnb.AmfN2Port,
		ranN2Port:           config.Gnb.RanN2Port,
		upfN3Port:           config.Gnb.UpfN3Port,
		ranN3Port:           config.Gnb.RanN3Port,
		ranControlPlanePort: config.Gnb.RanControlPlanePort,
		ranDataPlanePort:    config.Gnb.RanDataPlanePort,
		xnPort:              config.Gnb.XnPort,

		ngapPpid: config.Gnb.NgapPpid,

		gnbId:   gnbId,
		gnbName: config.Gnb.GnbName,

		plmnId: plmnId,
		tai:    tai,
		snssai: snssai,

		staticNrdc: config.Gnb.StaticNrdc,
		xnInterface: xnInterface{
			xnIp:   config.Gnb.XnInterface.XnIp,
			xnPort: config.Gnb.XnInterface.XnPort,
		},

		ranUeConns: sync.Map{},
		xnUeConns:  sync.Map{},
		teidToConn: sync.Map{},

		ranUeNgapIdGenerator: NewRanUeNgapIdGenerator(),
		teidGenerator:        NewTeidGenerator(),

		api: api{
			ip:   config.Gnb.Api.Ip,
			port: config.Gnb.Api.Port,

			router: nil,
			server: nil,
		},

		GnbLogger: gnbLogger,
	}
}

func (g *Gnb) Start(ctx context.Context) error {
	g.RanLog.Infoln("Starting GNB")

	if err := g.connectToAmf(); err != nil {
		g.SctpLog.Errorf("Error connecting to AMF: %v", err)
		return err
	}

	if err := g.setupN2(); err != nil {
		g.NgapLog.Errorf("Error setting up N2: %v", err)
		if err := g.n2Conn.Close(); err != nil {
			g.SctpLog.Errorf("Error closing N2 connection: %v", err)
		}
		return err
	}

	if err := g.connectToUpf(); err != nil {
		g.GtpLog.Errorf("Error connecting to UPF: %v", err)
		if err := g.n2Conn.Close(); err != nil {
			g.SctpLog.Errorf("Error closing N2 connection: %v", err)
		}
		return err
	}
	g.startGtpProcessor(ctx)

	if err := g.startXnListener(); err != nil {
		g.XnLog.Errorf("Error starting XN listener: %v", err)
		close(g.gtpChannel)
		if err := g.n3Conn.Close(); err != nil {
			g.GtpLog.Errorf("Error closing N3 connection: %v", err)
		}
		if err := g.n2Conn.Close(); err != nil {
			g.SctpLog.Errorf("Error closing N2 connection: %v", err)
		}
		return err
	}

	if err := g.startRanControlPlaneListener(); err != nil {
		g.RanLog.Errorf("Error starting ran control plane listener: %v", err)

		if err := (*g.xnListener).Close(); err != nil {
			g.XnLog.Errorf("Error closing XN listener: %v", err)
		}

		close(g.gtpChannel)
		if err := g.n3Conn.Close(); err != nil {
			g.GtpLog.Errorf("Error closing N3 connection: %v", err)
		}
		if err := g.n2Conn.Close(); err != nil {
			g.SctpLog.Errorf("Error closing N2 connection: %v", err)
		}
		return err
	}

	if err := g.startRanDataPlaneListener(); err != nil {
		g.RanLog.Errorf("Error starting ran data plane listener: %v", err)
		if err := (*g.ranControlPlaneListener).Close(); err != nil {
			g.RanLog.Errorf("Error closing ran control plane listener: %v", err)
		}
		if err := (*g.xnListener).Close(); err != nil {
			g.XnLog.Errorf("Error closing XN listener: %v", err)
		}
		close(g.gtpChannel)
		if err := g.n3Conn.Close(); err != nil {
			g.GtpLog.Errorf("Error closing N3 connection: %v", err)
		}
		if err := g.n2Conn.Close(); err != nil {
			g.SctpLog.Errorf("Error closing N2 connection: %v", err)
		}
		return err
	}

	go func() {
		for {
			conn, err := (*g.xnListener).Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				g.XnLog.Errorf("Error accepting XN connection: %v", err)
				continue
			}
			g.XnLog.Infof("New XN connection accepted from: %v", conn.RemoteAddr())
			go xnInterfaceProcessor(conn, g)
		}
	}()

	go func() {
		for {
			conn, err := (*g.ranControlPlaneListener).Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				g.RanLog.Errorf("Error accepting UE connection: %v", err)
				continue
			}
			g.RanLog.Infof("New UE connection accepted from: %v", conn.RemoteAddr())
			ranUe := NewRanUe(conn, g.ranUeNgapIdGenerator)
			if g.staticNrdc {
				ranUe.ActivateNrdc()
			}

			g.ranUeConns.Store(ranUe, struct{}{})
			go g.handleRanConnection(ctx, ranUe)
		}
	}()

	g.startApiServer()

	g.RanLog.Infoln("GNB started")
	return nil
}

func (g *Gnb) Stop() {
	g.RanLog.Infoln("Stopping GNB")

	g.stopApiServer()

	if err := (*g.ranDataPlaneListener).Close(); err != nil {
		g.RanLog.Errorf("Error stopping ran data plane listener: %v", err)
		return
	}
	g.RanLog.Debugln("ran data plane listener stopped")
	g.RanLog.Tracef("ran data plane listener stopped at %s:%d", g.ranDataPlaneIp, g.ranDataPlanePort)

	if err := (*g.ranControlPlaneListener).Close(); err != nil {
		g.RanLog.Errorf("Error stopping gNB: %v", err)
		return
	}
	g.RanLog.Debugln("gNB listener stopped")
	g.RanLog.Tracef("gNB listener stopped at %s:%d", g.ranControlPlaneIp, g.ranControlPlanePort)

	if err := (*g.xnListener).Close(); err != nil {
		g.XnLog.Errorf("Error closing XN listener: %v", err)
	}
	g.XnLog.Debugln("XN listener stopped")
	g.XnLog.Tracef("XN listener stopped at %s:%d", g.xnIp, g.xnPort)

	var wg sync.WaitGroup
	g.xnUeConns.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(xnUe *XnUe) {
			defer wg.Done()
			if xnUe, ok := key.(*XnUe); ok && xnUe.GetDataPlaneConn() != nil {
				g.XnLog.Tracef("XN UE %v still in connection", xnUe.GetDataPlaneConn().RemoteAddr())
				if err := xnUe.GetDataPlaneConn().Close(); err != nil {
					g.XnLog.Errorf("Error closing XN UE connection: %v", err)
				}
			}
			g.XnLog.Debugf("Closed XN UE connection from: %v", xnUe.GetDataPlaneConn().RemoteAddr())
		}(key.(*XnUe))
		return true
	})
	g.ranUeConns.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(ranUe *RanUe) {
			defer wg.Done()
			if ranUe, ok := key.(*RanUe); ok {
				g.RanLog.Tracef("UE %v still in connection", ranUe.GetN1Conn().RemoteAddr())
				if err := ranUe.GetN1Conn().Close(); err != nil {
					g.RanLog.Errorf("Error closing UE connection: %v", err)
				}
			}
			g.RanLog.Debugf("Closed UE connection from: %v", ranUe.GetN1Conn().RemoteAddr())
		}(key.(*RanUe))
		return true
	})
	wg.Wait()

	close(g.gtpChannel)
	g.GtpLog.Debugln("GTP channel closed")

	if err := g.n3Conn.Close(); err != nil {
		g.RanLog.Errorf("Error stopping N3 connection: %v", err)
		return
	}
	g.GtpLog.Tracef("N3 connection closed at %s:%d", g.ranN3Ip, g.ranN3Port)
	g.GtpLog.Debugln("N3 connection closed")

	if err := g.n2Conn.Close(); err != nil {
		g.SctpLog.Errorf("Error stopping N2 connection: %v", err)
		return
	}
	g.SctpLog.Tracef("N2 connection closed at %s:%d", g.ranN2Ip, g.ranN2Port)
	g.SctpLog.Debugln("N2 connection closed")

	g.RanLog.Infoln("GNB stopped")
}

func (g *Gnb) connectToAmf() error {
	g.RanLog.Infoln("Connecting to AMF")

	amfAddr, gnbAddr, err := getAmfAndGnbSctpN2Addr(g.amfN2Ip, g.ranN2Ip, g.amfN2Port, g.ranN2Port)
	if err != nil {
		return err
	}
	g.SctpLog.Tracef("AMF N2 address: %v", amfAddr.String())
	g.SctpLog.Tracef("GNB N2 address: %v", gnbAddr.String())

	conn, err := sctp.DialSCTP("sctp", gnbAddr, amfAddr)
	if err != nil {
		return fmt.Errorf("error connecting to AMF: %v", err)
	}
	g.SctpLog.Debugln("Dial SCTP to AMF success")

	info, err := conn.GetDefaultSentParam()
	if err != nil {
		return err
	}
	g.SctpLog.Tracef("N2 connection default sent param: %+v", info)

	info.PPID = g.ngapPpid
	if err := conn.SetDefaultSentParam(info); err != nil {
		return fmt.Errorf("error setting default sent param: %v", err)
	}

	g.n2Conn = conn

	g.RanLog.Infof("Connected to AMF: %v", amfAddr.String())
	return nil
}

func (g *Gnb) connectToUpf() error {
	g.RanLog.Infoln("Connecting to UPF")
	upfAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", g.upfN3Ip, g.upfN3Port))
	if err != nil {
		return fmt.Errorf("error resolving UPF N3 IP address: %v", err)
	}

	ranAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", g.ranN3Ip, g.ranN3Port))
	if err != nil {
		return fmt.Errorf("error resolving RAN N3 IP address: %v", err)
	}

	conn, err := net.DialUDP("udp", ranAddr, upfAddr)
	if err != nil {
		return fmt.Errorf("error connecting to UPF: %v", err)
	}
	g.GtpLog.Debugln("Dial UDP to UPF success")

	g.n3Conn = conn
	g.RanLog.Infof("Connected to UPF: %v, local: %v", upfAddr.String(), conn.LocalAddr().String())
	return nil
}

func (g *Gnb) setupN2() error {
	g.RanLog.Infoln("Setting up N2")

	request, err := getNgapSetupRequest(g.gnbId, g.gnbName, g.plmnId, g.tai, g.snssai)
	if err != nil {
		return fmt.Errorf("error getting NGAP setup request: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup request: %+v", request)

	n, err := g.n2Conn.Write(request)
	if err != nil {
		return fmt.Errorf("error sending NGAP setup request: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of NGAP setup request", n)
	g.NgapLog.Debugln("Sent NGAP setup request to AMF")

	responseRaw := make([]byte, 2048)
	n, err = g.n2Conn.Read(responseRaw)
	if err != nil {
		return fmt.Errorf("error reading NGAP setup response: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup responseRaw: %+v", responseRaw[:n])

	response, err := ngap.Decoder(responseRaw[:n])
	if err != nil {
		return fmt.Errorf("error decoding NGAP setup response: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup response: %+v", response)
	g.NgapLog.Debugln("Received NGAP setup response from AMF")

	if (response.Present != ngapType.NGAPPDUPresentSuccessfulOutcome) || (response.SuccessfulOutcome.ProcedureCode.Value != ngapType.ProcedureCodeNGSetup) {
		return fmt.Errorf("error NGAP setup response: %+v", response)
	}

	g.NgapLog.Infoln("============= gNB Info =============")

	g.NgapLog.Infof("gNB ID: %s, name: %s", hex.EncodeToString(g.gnbId), g.gnbName)

	plmnId := ngapConvert.PlmnIdToModels(g.plmnId)
	g.NgapLog.Infof("PLMN ID: %v", plmnId)

	tai := ngapConvert.TaiToModels(g.tai)
	g.NgapLog.Infof("TAC: %v, broadcast PLMN ID: %v", tai.Tac, tai.PlmnId)

	snssai := ngapConvert.SNssaiToModels(g.snssai)
	g.NgapLog.Infof("SST: %v, SD: %v", snssai.Sst, snssai.Sd)

	g.NgapLog.Infoln("====================================")

	g.RanLog.Infoln("N2 setup complete")
	return nil
}

func (g *Gnb) setupN1(ranUe *RanUe) error {
	g.RanLog.Infoln("Setting up N1")

	// ue initialization
	if err := g.processUeInitialization(ranUe); err != nil {
		return fmt.Errorf("error process ue initialization: %v", err)
	}
	time.Sleep(1 * time.Second)

	// pdu session establishment
	ranUe.SetDlTeid(g.teidGenerator.AllocateTeid())
	pduSessionResourceSetupRequestTransfer := ngapType.PDUSessionResourceSetupRequestTransfer{}
	if err := g.processUePduSessionEstablishment(ranUe, &pduSessionResourceSetupRequestTransfer); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	// accept UE data plane connection
	ueDataPlaneConn, err := (*g.ranDataPlaneListener).Accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			g.RanLog.Warnf("UE data plane connection acceptor closed: %v", err)
			return nil
		}
		return err
	}
	ranUe.SetDataPlaneConn(ueDataPlaneConn)
	g.RanLog.Infof("Accepted UE data plane connection from: %v", ueDataPlaneConn.RemoteAddr())

	// configure UE mapping teid to conn
	for _, item := range pduSessionResourceSetupRequestTransfer.ProtocolIEs.List {
		switch item.Id.Value {
		case ngapType.ProtocolIEIDPDUSessionAggregateMaximumBitRate:
		case ngapType.ProtocolIEIDULNGUUPTNLInformation:
			ranUe.SetUlTeid(item.Value.ULNGUUPTNLInformation.GTPTunnel.GTPTEID.Value)
		case ngapType.ProtocolIEIDAdditionalULNGUUPTNLInformation:
		case ngapType.ProtocolIEIDPDUSessionType:
		case ngapType.ProtocolIEIDQosFlowSetupRequestList:
		}
	}
	g.teidToConn.Store(hex.EncodeToString(ranUe.GetDlTeid()), ueDataPlaneConn)
	g.GtpLog.Debugf("Stored UE %s data plane connection with teid %s to teidToConn", ranUe.GetMobileIdentityIMSI(), hex.EncodeToString(ranUe.GetDlTeid()))

	g.RanLog.Infof("UE %s N1 setup complete", ranUe.GetMobileIdentityIMSI())
	return nil
}

func (g *Gnb) releaseN1(ranUe *RanUe) error {
	g.RanLog.Infoln("Waiting for UE to release N1")

	if err := g.processUeDeRegistration(ranUe); err != nil {
		return fmt.Errorf("error processing UE deregistration: %v", err)
	}

	g.RanLog.Infoln("N1 released")
	return nil
}

func (g *Gnb) startGtpProcessor(ctx context.Context) {
	g.GtpLog.Infoln("Starting GTP processor")

	g.gtpChannel = make(chan []byte)

	go forwardGtpPacketToN3Conn(ctx, g.n3Conn, g.gtpChannel, g.GnbLogger)
	g.GtpLog.Debugln("Forward GTP packet to N3 connection started")

	go receiveGtpPacketFromN3Conn(ctx, g.n3Conn, g.GnbLogger, &g.teidToConn)
	g.GtpLog.Debugln("Receive GTP packet from N3 connection started")

	g.GtpLog.Infoln("GTP processor started")
}

func (g *Gnb) startXnListener() error {
	g.XnLog.Infoln("Starting XN listener")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.xnIp, g.xnPort))
	if err != nil {
		return err
	}
	g.xnListener = &listener

	g.XnLog.Infoln("============= XN Info ==============")
	g.XnLog.Infof("XN access address: %s:%d", g.xnIp, g.xnPort)
	g.XnLog.Infoln("====================================")

	g.XnLog.Infoln("XN listener started")
	return nil
}

func (g *Gnb) startRanControlPlaneListener() error {
	g.RanLog.Infoln("Starting RAN control plane listener")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.ranControlPlaneIp, g.ranControlPlanePort))
	if err != nil {
		return err
	}
	g.ranControlPlaneListener = &listener

	g.RanLog.Infoln("====== RAN Control Plane Info ======")
	g.RanLog.Infof("RAN Control Plane access address: %s:%d", g.ranControlPlaneIp, g.ranControlPlanePort)
	g.RanLog.Infoln("====================================")

	g.RanLog.Infoln("RAN control plane listener started")
	return nil
}

func (g *Gnb) startRanDataPlaneListener() error {
	g.RanLog.Infoln("Starting RAN data plane listener")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.ranDataPlaneIp, g.ranDataPlanePort))
	if err != nil {
		return err
	}
	g.ranDataPlaneListener = &listener

	g.RanLog.Infoln("======= RAN Data Plane Info ========")
	g.RanLog.Infof("RAN Data Plane access address: %s:%d", g.ranDataPlaneIp, g.ranDataPlanePort)
	g.RanLog.Infoln("====================================")

	g.RanLog.Infoln("RAN data plane listener started")
	return nil
}

func (g *Gnb) handleRanConnection(ctx context.Context, ranUe *RanUe) {
	defer func() {
		if err := ranUe.GetN1Conn().Close(); err != nil {
			g.RanLog.Errorf("Error closing UE connection: %v", err)
		}
		g.RanLog.Infof("Closed UE connection from: %v", ranUe.GetN1Conn().RemoteAddr())
		ranUe.Release(g.ranUeNgapIdGenerator, g.teidGenerator)
		g.ranUeConns.Delete(ranUe)
	}()

	if err := g.setupN1(ranUe); err != nil {
		g.RanLog.Errorf("Error setting up N1: %v", err)
		return
	}
	g.GtpLog.Debugf("DL TEID: %s, UL TEID: %s", hex.EncodeToString(ranUe.GetDlTeid()), hex.EncodeToString(ranUe.GetUlTeid()))

	// handle data plane from UE
	go g.startUeDataPlaneProcessor(ranUe.GetDataPlaneConn(), ranUe.GetUlTeid(), ranUe.GetDlTeid(), false, ranUe.GetMobileIdentityIMSI())

	if err := g.releaseN1(ranUe); err != nil {
		g.RanLog.Errorf("Error releasing N1: %v", err)
		return
	}
	g.RanLog.Infof("UE %s N1 released", ranUe.GetMobileIdentityIMSI())
}

func (g *Gnb) startUeDataPlaneProcessor(ueDataPlaneConn net.Conn, ulTeid, dlTeid aper.OctetString, isXnUe bool, imsi string) {
	buffer := make([]byte, 4096)
	for {
		n, err := ueDataPlaneConn.Read(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				g.teidToConn.Delete(hex.EncodeToString(dlTeid))
				g.GtpLog.Debugf("Deleted teid %s from teidToConn", hex.EncodeToString(dlTeid))
				g.GtpLog.Infof("Connection from UE IP: %v closed", ueDataPlaneConn.RemoteAddr())

				if isXnUe {
					g.xnUeConns.Range(func(key, value interface{}) bool {
						if key.(*XnUe).GetIMSI() == imsi {
							key.(*XnUe).Release(g.teidGenerator)
							g.xnUeConns.Delete(key)
							g.XnLog.Debugf("Deleted XN UE %s from xnUeConns", imsi)
						}
						return true
					})
					g.XnLog.Debugf("Deleted XN UE %s from xnUeConns", imsi)
				}
				return
			}
			g.RanLog.Warnf("Error reading from UE connection: %v", err)
		}
		g.RanLog.Tracef("Received %d bytes of data from UE: %+v", n, buffer[:n])
		g.RanLog.Tracef("Received %d bytes of data from UE", n)

		tmp := make([]byte, n)
		copy(tmp, buffer[:n])
		go formatGtpPacketAndWriteToGtpChannel(ulTeid, tmp, g.gtpChannel, g.GnbLogger)
	}
}

func (g *Gnb) processUeInitialization(ranUe *RanUe) error {
	g.RanLog.Infoln("Processing UE initialization")

	// receive ue registration request from UE and send to AMF
	ueRegistrationRequest := make([]byte, 1024)
	n, err := ranUe.GetN1Conn().Read(ueRegistrationRequest)
	if err != nil {
		return fmt.Errorf("error receive ue registration request from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of UE registration request from UE", n)

	nasMessage := nas.NewMessage()
	if err := nasMessage.GmmMessageDecode(&ueRegistrationRequest); err != nil {
		return fmt.Errorf("error decode ue registration request from UE: %v", err)
	}
	ranUe.SetMobileIdentity5GS(nasMessage.GmmMessage.RegistrationRequest.MobileIdentity5GS)
	g.NasLog.Debugf("Receive UE %s registration request from UE", ranUe.GetMobileIdentityIMSI())

	ueInitialMessage, err := getInitialUeMessage(ranUe.GetRanUeId(), ueRegistrationRequest, g.plmnId, g.tai)
	if err != nil {
		return fmt.Errorf("error get initial ue message: %v", err)
	}
	g.NgapLog.Tracef("Get initial UE message: %+v", ueInitialMessage)

	if n, err = g.n2Conn.Write(ueInitialMessage); err != nil {
		return fmt.Errorf("error send initial ue message to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of initial UE message to AMF", n)
	g.NgapLog.Debugln("Sent initial UE message to AMF")

	// receive nas authentication request from AMF and send to UE
	ngapNasAuthenticationRequestRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapNasAuthenticationRequestRaw)
	if err != nil {
		return fmt.Errorf("error receive initial ue response from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NAS Authentication Request from AMF", n)
	g.NgapLog.Debugln("Receive NAS Authentication Request from AMF")

	ngapNasAuthenticationRequest, err := ngap.Decoder(ngapNasAuthenticationRequestRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas authentication request from AMF: %v", err)
	}
	if ngapNasAuthenticationRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapNasAuthenticationRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return fmt.Errorf("error NGAP nas authentication request: %+v", ngapNasAuthenticationRequest)
	}
	g.NgapLog.Tracef("NGAP nas authentication request: %+v", ngapNasAuthenticationRequest)

	var nasAuthenticationRequest []byte
	for _, ie := range ngapNasAuthenticationRequest.InitiatingMessage.Value.DownlinkNASTransport.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			ranUe.SetAmfUeId(ie.Value.AMFUENGAPID.Value)
			g.NgapLog.Tracef("Set AMF UE ID: %d", ranUe.GetAmfUeId())
		case ngapType.ProtocolIEIDRANUENGAPID:
			ranUe.SetRanUeId(ie.Value.RANUENGAPID.Value)
			g.NgapLog.Tracef("Set RAN UE ID: %d", ranUe.GetRanUeId())
		case ngapType.ProtocolIEIDNASPDU:
			if ie.Value.NASPDU == nil {
				return fmt.Errorf("error NGAP nas authentication request: NASPDU is nil")
			}
			nasAuthenticationRequest = make([]byte, len(ie.Value.NASPDU.Value))
			copy(nasAuthenticationRequest, ie.Value.NASPDU.Value)
			g.NgapLog.Tracef("Get NASPDU: %+v", nasAuthenticationRequest)
		}
	}

	n, err = ranUe.GetN1Conn().Write(nasAuthenticationRequest)
	if err != nil {
		return fmt.Errorf("error send nas authentication request to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS Authentication Request to UE", n)
	g.NasLog.Debugln("Send NAS Authentication Request to UE")

	// receive nas authentication response from UE and send to AMF
	nasAuthenticationResponse := make([]byte, 1024)
	n, err = ranUe.GetN1Conn().Read(nasAuthenticationResponse)
	if err != nil {
		return fmt.Errorf("error receive nas authentication response from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of NAS Authentication Response from UE", n)
	g.NasLog.Debugln("Receive NAS Authentication Response from UE")

	uplinkNasTransport, err := getUplinkNasTransport(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), g.plmnId, g.tai, nasAuthenticationResponse[:n])
	if err != nil {
		return fmt.Errorf("error get uplink nas transport: %v", err)
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return fmt.Errorf("error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Sent uplink NAS transport to AMF")

	// receive nas security mode command message from AMF and send to UE
	ngapNasSecurityModeCommandRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapNasSecurityModeCommandRaw)
	if err != nil {
		return fmt.Errorf("error receive nas security mode command from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NAS Security Mode Command from AMF", n)
	g.NgapLog.Debugf("Receive NAS Security Mode Command from AMF")

	ngapNasSecurityModeCommand, err := ngap.Decoder(ngapNasSecurityModeCommandRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas security mode command from AMF: %v", err)
	}
	if ngapNasSecurityModeCommand.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapNasSecurityModeCommand.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return fmt.Errorf("error NGAP nas security mode command: %+v", ngapNasSecurityModeCommand)
	}
	g.NgapLog.Tracef("NGAP nas security mode command: %+v", ngapNasSecurityModeCommand)

	var nasSecurityModeCommand []byte
	for _, ie := range ngapNasSecurityModeCommand.InitiatingMessage.Value.DownlinkNASTransport.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDNASPDU:
			if ie.Value.NASPDU == nil {
				return fmt.Errorf("error NGAP nas security mode command: NASPDU is nil")
			}
			nasSecurityModeCommand = make([]byte, len(ie.Value.NASPDU.Value))
			copy(nasSecurityModeCommand, ie.Value.NASPDU.Value)
			g.NgapLog.Tracef("Get NASPDU: %+v", nasSecurityModeCommand)
		}
	}

	if n, err = ranUe.GetN1Conn().Write(nasSecurityModeCommand); err != nil {
		return fmt.Errorf("error send nas security mode command to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS Security Mode Command to UE", n)
	g.NasLog.Debugln("Send NAS Security Mode Command to UE")

	// receive nas security mode complete message from UE and send to AMF
	nasSecurityModeComplete := make([]byte, 1024)
	n, err = ranUe.GetN1Conn().Read(nasSecurityModeComplete)
	if err != nil {
		return fmt.Errorf("error receive nas security mode complete from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of NAS Security Mode Complete from UE", n)
	g.NasLog.Debugln("Receive NAS Security Mode Complete from UE")

	uplinkNasTransport, err = getUplinkNasTransport(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), g.plmnId, g.tai, nasSecurityModeComplete[:n])
	if err != nil {
		return fmt.Errorf("error get uplink nas transport: %v", err)
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return fmt.Errorf("error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Sent uplink NAS transport to AMF")

	// receive ngap initial context setup request from AMF
	ngapInitialContextSetupRequestRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapInitialContextSetupRequestRaw)
	if err != nil {
		return fmt.Errorf("error receive ngap initial context setup request from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NGAP Initial Context Setup Request from AMF", n)

	ngapInitialContextSetupRequest, err := ngap.Decoder(ngapInitialContextSetupRequestRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ngap initial context setup request from AMF: %v", err)
	}
	if ngapInitialContextSetupRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapInitialContextSetupRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeInitialContextSetup {
		return fmt.Errorf("error ngap initial context setup request: no initial context setup request")
	}
	g.NgapLog.Tracef("NGAP Initial Context Setup Request: %+v", ngapInitialContextSetupRequest)
	g.NgapLog.Debugln("Receive NGAP Initial Context Setup Request from AMF")

	// send ngap initial context setup response to AMF
	ngapInitialContextSetupResponse, err := getNgapInitialContextSetupResponse(ranUe.GetAmfUeId(), ranUe.GetRanUeId())
	if err != nil {
		return fmt.Errorf("error get ngap initial context setup response: %v", err)
	}
	g.NgapLog.Tracef("Get NGAP Initial Context Setup Response: %+v", ngapInitialContextSetupResponse)

	n, err = g.n2Conn.Write(ngapInitialContextSetupResponse)
	if err != nil {
		return fmt.Errorf("error send ngap initial context setup response to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of NGAP Initial Context Setup Response to AMF", n)
	g.NgapLog.Debugln("Send NGAP Initial Context Setup Response to AMF")

	// receive nas registration complete message from UE and send to AMF
	nasRegistrationComplete := make([]byte, 1024)
	n, err = ranUe.GetN1Conn().Read(nasRegistrationComplete)
	if err != nil {
		return fmt.Errorf("error receive nas registration complete from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of NAS Registration Complete from UE", n)
	g.NasLog.Debugln("Receive NAS Registration Complete from UE")

	uplinkNasTransport, err = getUplinkNasTransport(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), g.plmnId, g.tai, nasRegistrationComplete[:n])
	if err != nil {
		return fmt.Errorf("error get uplink nas transport: %v", err)
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return fmt.Errorf("error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Send NAS Registration Complete to AMF")

	// receive ue configuration update command message from AMF
	ueConfigurationUpdateCommandRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ueConfigurationUpdateCommandRaw)
	if err != nil {
		return fmt.Errorf("error receive ue configuration update command from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of UE Configuration Update Command from AMF", n)

	ueConfigurationUpdateCommand, err := ngap.Decoder(ueConfigurationUpdateCommandRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ue configuration update command from AMF: %v", err)
	}
	if ueConfigurationUpdateCommand.Present != ngapType.NGAPPDUPresentInitiatingMessage || ueConfigurationUpdateCommand.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return fmt.Errorf("error ue configuration update command: no ue configuration update command")
	}
	g.NgapLog.Tracef("UE Configuration Update Command: %+v", ueConfigurationUpdateCommand)
	g.NgapLog.Debugln("Receive UE Configuration Update Command from AMF")

	g.RanLog.Infof("UE %s initialized", ranUe.GetMobileIdentityIMSI())
	return nil
}

func (g *Gnb) processUePduSessionEstablishment(ranUe *RanUe, pduSessionResourceSetupRequestTransfer *ngapType.PDUSessionResourceSetupRequestTransfer) error {
	g.NgapLog.Infof("Processing UE %s PDU session establishment", ranUe.GetMobileIdentityIMSI())

	// receive pdu session establishment request from UE and send to AMF
	pduSessionEstablishmentRequest := make([]byte, 1024)
	n, err := ranUe.GetN1Conn().Read(pduSessionEstablishmentRequest)
	if err != nil {
		return fmt.Errorf("error receive pdu session establishment request from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of PDU Session Establishment Request from UE", n)
	g.NasLog.Debugln("Receive PDU Session Establishment Request from UE")

	uplinkNasTransport, err := getUplinkNasTransport(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), g.plmnId, g.tai, pduSessionEstablishmentRequest[:n])
	if err != nil {
		return fmt.Errorf("error get uplink nas transport: %v", err)
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return fmt.Errorf("error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Send PDU Session Establishment Request to AMF")

	// receive ngap pdu session resource setup request from AMF
	ngapPduSessionResourceSetupRequestRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapPduSessionResourceSetupRequestRaw)
	if err != nil {
		return fmt.Errorf("error receive ngap pdu session resource setup request from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NGAP PDU Session Resource Setup Request from AMF", n)
	g.NgapLog.Debugln("Receive NGAP PDU Session Resource Setup Request from AMF")

	ngapPduSessionResourceSetupRequest, err := ngap.Decoder(ngapPduSessionResourceSetupRequestRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ngap pdu session resource setup request from AMF: %v", err)
	}
	if ngapPduSessionResourceSetupRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapPduSessionResourceSetupRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodePDUSessionResourceSetup {
		return fmt.Errorf("error ngap pdu session resource setup request: no pdu session resource setup request")
	}
	g.NgapLog.Tracef("NGAP PDU Session Resource Setup Request: %+v", ngapPduSessionResourceSetupRequest)

	var nasPduSessionEstablishmentAccept []byte
	for _, ie := range ngapPduSessionResourceSetupRequest.InitiatingMessage.Value.PDUSessionResourceSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq:
			for _, pduSessionResourceSetupItem := range ie.Value.PDUSessionResourceSetupListSUReq.List {
				nasPduSessionEstablishmentAccept = make([]byte, len(pduSessionResourceSetupItem.PDUSessionNASPDU.Value))
				copy(nasPduSessionEstablishmentAccept, pduSessionResourceSetupItem.PDUSessionNASPDU.Value)
				g.NgapLog.Tracef("Get NASPDU: %+v", nasPduSessionEstablishmentAccept)

				if err := aper.UnmarshalWithParams(pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer, pduSessionResourceSetupRequestTransfer, "valueExt"); err != nil {
					return fmt.Errorf("error unmarshal pdu session resource setup request transfer: %v", err)
				}
				g.NgapLog.Tracef("Get PDUSessionResourceSetupRequestTransfer: %+v", pduSessionResourceSetupRequestTransfer)
			}
		case ngapType.ProtocolIEIDUEAggregateMaximumBitRate:
		}
	}

	var qosFlowPerTNLInformationItem ngapType.QosFlowPerTNLInformationItem
	if ranUe.IsNrdcActivated() {
		if qosFlowPerTNLInformationItem, err = g.xnPduSessionResourceSetupRequestTransfer(ranUe.GetMobileIdentityIMSI(), ngapPduSessionResourceSetupRequestRaw[:n]); err != nil {
			g.XnLog.Warnf("Error xn pdu session resource setup request transfer: %v", err)
		}
	}

	n, err = ranUe.GetN1Conn().Write(nasPduSessionEstablishmentAccept)
	if err != nil {
		return fmt.Errorf("error send nas pdu session establishment accept to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS PDU Session Establishment Accept to UE", n)
	g.NasLog.Debugln("Send NAS PDU Session Establishment Accept to UE")

	// send ngap pdu session resource setup response to AMF
	ngapPduSessionResourceSetupResponseTransfer, err := getPduSessionResourceSetupResponseTransfer(ranUe.GetDlTeid(), g.ranN3Ip, 1, g.staticNrdc, qosFlowPerTNLInformationItem)
	if err != nil {
		return fmt.Errorf("error get pdu session resource setup response transfer: %v", err)
	}
	g.NgapLog.Tracef("Get pdu session resource setup response transfer: %+v", ngapPduSessionResourceSetupResponseTransfer)

	ngapPduSessionResourceSetupResponse, err := getPduSessionResourceSetupResponse(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), 4, ngapPduSessionResourceSetupResponseTransfer)
	if err != nil {
		return fmt.Errorf("error get pdu session resource setup response: %v", err)
	}
	g.NgapLog.Tracef("Get pdu session resource setup response: %+v", ngapPduSessionResourceSetupResponse)

	n, err = g.n2Conn.Write(ngapPduSessionResourceSetupResponse)
	if err != nil {
		return fmt.Errorf("error send pdu session resource setup response to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of pdu session resource setup response to AMF", n)
	g.NgapLog.Debugln("Send PDU Session Resource Setup Response to AMF")

	g.NgapLog.Infof("UE %s PDU session establishment completed", ranUe.GetMobileIdentityIMSI())
	return nil
}

func (g *Gnb) processUePduSessionModifyIndication(ranUe *RanUe) error {
	g.NgapLog.Infoln("Processing UE PDU Session Modify Indication")

	pduSessionModifyIndicationTransfer, err := getPDUSessionResourceModifyIndicationTransfer(ranUe.GetDlTeid(), g.ranN3Ip, 1)
	if err != nil {
		return fmt.Errorf("error get pdu session modify indication transfer: %v", err)
	}
	g.NgapLog.Tracef("Get pdu session modify indication transfer: %+v", pduSessionModifyIndicationTransfer)

	// send ngap pdu session resource modify indication to AMF
	pduSessionModifyIndication, err := getPDUSessionResourceModifyIndication(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), 4, pduSessionModifyIndicationTransfer)
	if err != nil {
		return fmt.Errorf("error get pdu session modify indication: %v", err)
	}
	g.NgapLog.Tracef("Get pdu session modify indication: %+v", pduSessionModifyIndication)

	if pduSessionModifyIndication, err = g.xnPduSessionResourceModifyIndication(ranUe.GetMobileIdentityIMSI(), pduSessionModifyIndication); err != nil {
		g.XnLog.Errorf("Error xn pdu session resource modify indication: %v", err)
		return fmt.Errorf("error xn pdu session resource modify indication: %v", err)
	}
	g.XnLog.Tracef("Get pdu session modify indication: %+v", pduSessionModifyIndication)

	n, err := g.n2Conn.Write(pduSessionModifyIndication)
	if err != nil {
		return fmt.Errorf("error send pdu session modify indication to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of pdu session modify indication to AMF", n)
	g.NgapLog.Debugln("Send PDU Session Modify Indication to AMF")

	// receive ngap pdu session resource setup request from AMF
	ngapPduSessionResourceModifyConfirmRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapPduSessionResourceModifyConfirmRaw)
	if err != nil {
		return fmt.Errorf("error receive ngap pdu session resource modify confirm from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NGAP PDU Session Resource Modify Confirm from AMF", n)
	g.NgapLog.Debugln("Receive NGAP PDU Session Resource Modify Confirm from AMF")

	ngapPduSessionResourceModifyConfirm, err := ngap.Decoder(ngapPduSessionResourceModifyConfirmRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ngap pdu session resource modify confirm from AMF: %v", err)
	}
	if ngapPduSessionResourceModifyConfirm.Present != ngapType.NGAPPDUPresentSuccessfulOutcome || ngapPduSessionResourceModifyConfirm.SuccessfulOutcome.ProcedureCode.Value != ngapType.ProcedureCodePDUSessionResourceModifyIndication {
		return fmt.Errorf("error ngap pdu session resource modify confirm: no pdu session resource modify confirm")
	}
	g.NgapLog.Tracef("NGAP PDU Session Resource Modify Confirm: %+v", ngapPduSessionResourceModifyConfirm)

	// check successful outcome
	for _, ie := range ngapPduSessionResourceModifyConfirm.SuccessfulOutcome.Value.PDUSessionResourceModifyConfirm.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDPDUSessionResourceModifyListModCfm:
			g.NgapLog.Infoln("PDU session modify indication successful")
		case ngapType.ProtocolIEIDPDUSessionResourceFailedToModifyListModCfm:
			return fmt.Errorf("error ngap pdu session resource modify confirm: pdu session resource modify failed")
		}
	}

	// send confirm to Xm for update xnUE ULTEID
	if !ranUe.IsNrdcActivated() {
		if _, err = g.xnPduSessionResourceModifyConfirm(ranUe.GetMobileIdentityIMSI(), ngapPduSessionResourceModifyConfirmRaw[:n]); err != nil {
			g.XnLog.Errorf("Error xn pdu session resource modify confirm: %v", err)
			return fmt.Errorf("error xn pdu session resource modify confirm: %v", err)
		}
		g.XnLog.Debugln("XN PDU Session Resource Modify Confirm sent")
	}

	// update ranUe NRDC status
	if ranUe.IsNrdcActivated() {
		ranUe.DeactivateNrdc()
		g.NgapLog.Infof("UE %s NRDC deactivated", ranUe.GetMobileIdentityIMSI())
	} else {
		ranUe.ActivateNrdc()
		g.NgapLog.Infof("UE %s NRDC activated", ranUe.GetMobileIdentityIMSI())
	}

	g.NgapLog.Infof("UE %s PDU session modify indication completed", ranUe.GetMobileIdentityIMSI())
	return nil
}

func (g *Gnb) processUeDeRegistration(ranUe *RanUe) error {
	g.RanLog.Infoln("Waiting for UE to deregister")

	// receive ue deregistration request from UE and send to AMF
	ueDeRegistrationRequest := make([]byte, 1024)
	n, err := ranUe.GetN1Conn().Read(ueDeRegistrationRequest)
	if err != nil {
		return fmt.Errorf("error reading from UE connection: %v", err)
	}
	g.RanLog.Tracef("Received %d bytes of UE deregistration request from UE: %+v", n, ueDeRegistrationRequest[:n])
	g.RanLog.Tracef("Received %d bytes of UE deregistration request from UE", n)

	uplinkNasTransport, err := getUplinkNasTransport(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), g.plmnId, g.tai, ueDeRegistrationRequest[:n])
	if err != nil {
		return fmt.Errorf("error get uplink nas transport: %v", err)
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return fmt.Errorf("error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Send UE deregistration request to AMF")

	// receive ue deregistration accept from AMF
	ngapUeDeRegistrationAcceptRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapUeDeRegistrationAcceptRaw)
	if err != nil {
		return fmt.Errorf("error receive ue deregistration accept from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of UE deregistration accept from AMF", n)
	g.NgapLog.Debugln("Receive UE deregistration accept from AMF")

	ngapUeDeRegistrationAccept, err := ngap.Decoder(ngapUeDeRegistrationAcceptRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ue deregistration accept from AMF: %v", err)
	}
	if ngapUeDeRegistrationAccept.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapUeDeRegistrationAccept.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return fmt.Errorf("error NGAP ue deregistration accept: %+v", ngapUeDeRegistrationAccept)
	}
	g.NgapLog.Tracef("NGAP UE deregistration accept: %+v", ngapUeDeRegistrationAccept)

	var nasUeDeRegistrationAccept []byte
	for _, ie := range ngapUeDeRegistrationAccept.InitiatingMessage.Value.DownlinkNASTransport.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDNASPDU:
			if ie.Value.NASPDU == nil {
				return fmt.Errorf("error NGAP ue deregistration accept: NASPDU is nil")
			}
			nasUeDeRegistrationAccept = make([]byte, len(ie.Value.NASPDU.Value))
			copy(nasUeDeRegistrationAccept, ie.Value.NASPDU.Value)
			g.NgapLog.Tracef("Get NASPDU: %+v", nasUeDeRegistrationAccept)
		}
	}

	n, err = ranUe.GetN1Conn().Write(nasUeDeRegistrationAccept)
	if err != nil {
		return fmt.Errorf("error send nas ue deregistration accept to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS UE deregistration Accept to UE", n)
	g.NasLog.Debugln("Send NAS UE deregistration Accept to UE")

	// receive ngap ue context release command from AMF
	ngapUeContextReleaseCommandRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapUeContextReleaseCommandRaw)
	if err != nil {
		return fmt.Errorf("error receive ngap ue context release command from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NGAP UE Context Release Command from AMF", n)

	ngapUeContextReleaseCommand, err := ngap.Decoder(ngapUeContextReleaseCommandRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ngap ue context release command from AMF: %v", err)
	}
	if ngapUeContextReleaseCommand.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapUeContextReleaseCommand.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeUEContextRelease {
		return fmt.Errorf("error ngap ue context release command: %+v", ngapUeContextReleaseCommand)
	}
	g.NgapLog.Tracef("NGAP UE Context Release Command: %+v", ngapUeContextReleaseCommand)
	g.NgapLog.Debugln("Receive NGAP UE Context Release Command from AMF")

	// send ngap ue context release complete to AMF
	ngapUeContextReleaseCompleteMessage, err := getNgapUeContextReleaseCompleteMessage(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), []int64{4}, g.plmnId, g.tai)
	if err != nil {
		return fmt.Errorf("error get ngap ue context release complete message: %v", err)
	}
	g.NgapLog.Tracef("Get NGAP UE Context Release Complete Message: %+v", ngapUeContextReleaseCompleteMessage)

	n, err = g.n2Conn.Write(ngapUeContextReleaseCompleteMessage)
	if err != nil {
		return fmt.Errorf("error send ngap ue context release complete message to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of NGAP UE Context Release Complete Message to AMF", n)
	g.NgapLog.Debugln("Send NGAP UE Context Release Complete Message to AMF")

	g.RanLog.Infoln("UE deregistration complete")
	return nil
}

func (g *Gnb) xnPduSessionResourceSetupRequestTransfer(imsi string, ngapPduSessionResourceSetupRequestRaw []byte) (ngapType.QosFlowPerTNLInformationItem, error) {
	g.XnLog.Infoln("Processing XN PDU Session Resource Setup Request Transfer")

	var qosFlowPerTNLInformationItem ngapType.QosFlowPerTNLInformationItem

	xnConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", g.xnInterface.xnIp, g.xnInterface.xnPort))
	if err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error dial xn: %v", err)
	}
	g.XnLog.Debugf("Dial XN at %s:%d", g.xnIp, g.xnPort)

	xnPdu := NewXnPdu(imsi, ngapPduSessionResourceSetupRequestRaw)
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error marshal xn pdu: %v", err)
	}

	n, err := xnConn.Write(xnPduBytes)
	if err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error send ngap pdu session resource setup request to xn: %v", err)
	}
	g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Setup Request to XN", n)
	g.XnLog.Debugln("Send NGAP PDU Session Resource Setup Request to XN")

	if err = xnConn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error set read deadline: %v", err)
	}
	buffer := make([]byte, 4096)
	n, err = xnConn.Read(buffer)
	if err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error read ngap pdu session resource setup response from xn: %v", err)
	}
	g.XnLog.Tracef("Received %d bytes of NGAP PDU Session Resource Setup Response from XN", n)
	g.XnLog.Debugln("Receive NGAP PDU Session Resource Setup Response from XN")

	xnPdu = &XnPdu{}
	if err := xnPdu.Unmarshal(buffer[:n]); err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error unmarshal xn pdu: %v", err)
	}
	g.XnLog.Tracef("Received XN PDU: %+v", xnPdu)
	g.XnLog.Debugln("Receive XN PDU")

	if err := aper.UnmarshalWithParams(xnPdu.Data, &qosFlowPerTNLInformationItem, "valueExt"); err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error unmarshal qos flow per tnl information item: %v", err)
	}
	g.XnLog.Tracef("Get QoS Flow per TNL Information Item: %+v", qosFlowPerTNLInformationItem)

	if err := xnConn.Close(); err != nil {
		return qosFlowPerTNLInformationItem, fmt.Errorf("error close xn connection: %v", err)
	}

	g.XnLog.Infoln("XN PDU Session Resource Setup Request Transfer completed")
	return qosFlowPerTNLInformationItem, nil
}

func (g *Gnb) xnPduSessionResourceModifyIndication(imsi string, ngapPduSessionResourceModifyIndicationRaw []byte) ([]byte, error) {
	g.XnLog.Infoln("Processing XN PDU Session Resource Modify Indication Transfer")

	xnConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", g.xnInterface.xnIp, g.xnInterface.xnPort))
	if err != nil {
		return nil, fmt.Errorf("error dial xn: %v", err)
	}
	g.XnLog.Debugf("Dial XN at %s:%d", g.xnIp, g.xnPort)

	xnPdu := NewXnPdu(imsi, ngapPduSessionResourceModifyIndicationRaw)
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		return nil, fmt.Errorf("error marshal xn pdu: %v", err)
	}

	n, err := xnConn.Write(xnPduBytes)
	if err != nil {
		return nil, fmt.Errorf("error send ngap pdu session resource modify indication transfer to xn: %v", err)
	}
	g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Modify Indication Transfer to XN", n)
	g.XnLog.Debugln("Send NGAP PDU Session Resource Modify Indication Transfer to XN")

	// if the modify is from 2 -> 1, here will read the same pdu as the request
	// if the modify is from 1 -> 2, here will read the appended pdu with secondary tunnel information
	if err = xnConn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return nil, fmt.Errorf("error set read deadline: %v", err)
	}
	buffer := make([]byte, 4096)
	n, err = xnConn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("error read ngap pdu session resource modify indication response from xn: %v", err)
	}
	g.XnLog.Tracef("Received %d bytes of NGAP PDU Session Resource Modify Indication Response from XN", n)
	g.XnLog.Debugln("Receive NGAP PDU Session Resource Modify Indication Response from XN")

	xnPdu = &XnPdu{}
	if err := xnPdu.Unmarshal(buffer[:n]); err != nil {
		return nil, fmt.Errorf("error unmarshal xn pdu: %v", err)
	}
	g.XnLog.Tracef("Received XN PDU: %+v", xnPdu)
	g.XnLog.Debugln("Receive XN PDU")

	if err := xnConn.Close(); err != nil {
		return xnPdu.Data, fmt.Errorf("error close xn connection: %v", err)
	}

	g.XnLog.Infoln("XN PDU Session Resource Modify Indication Transfer completed")
	return xnPdu.Data, nil
}

func (g *Gnb) xnPduSessionResourceModifyConfirm(imsi string, ngapPduSessionResourceModifyConfirmRaw []byte) ([]byte, error) {
	g.XnLog.Infoln("Processing XN PDU Session Resource Modify Confirm")

	xnConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", g.xnInterface.xnIp, g.xnInterface.xnPort))
	if err != nil {
		return nil, fmt.Errorf("error dial xn: %v", err)
	}
	g.XnLog.Debugf("Dial XN at %s:%d", g.xnIp, g.xnPort)

	xnPdu := NewXnPdu(imsi, ngapPduSessionResourceModifyConfirmRaw)
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		return nil, fmt.Errorf("error marshal xn pdu: %v", err)
	}

	n, err := xnConn.Write(xnPduBytes)
	if err != nil {
		return nil, fmt.Errorf("error send ngap pdu session resource modify confirm to xn: %v", err)
	}
	g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Modify Confirm to XN", n)
	g.XnLog.Debugln("Send NGAP PDU Session Resource Modify Confirm to XN")

	if err = xnConn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return nil, fmt.Errorf("error set read deadline: %v", err)
	}
	buffer := make([]byte, 4096)
	n, err = xnConn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("error read ngap pdu session resource modify confirm response from xn: %v", err)
	}
	g.XnLog.Tracef("Received %d bytes of NGAP PDU Session Resource Modify Confirm Response from XN", n)
	g.XnLog.Debugln("Receive NGAP PDU Session Resource Modify Confirm Response from XN")

	if err := xnConn.Close(); err != nil {
		return nil, fmt.Errorf("error close xn connection: %v", err)
	}

	g.XnLog.Infoln("XN PDU Session Resource Modify Confirm completed")
	return nil, nil
}

func (g *Gnb) startApiServer() {
	g.ApiLog.Infoln("Starting API server")

	g.api.router = util.NewGinRouter(util.GNB_API_PREFIX, g.initApiRoutes())

	g.api.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", g.api.ip, g.api.port),
		Handler: g.api.router,
	}

	go func() {
		if err := g.api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			g.ApiLog.Errorf("Failed to start API server: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	g.ApiLog.Infoln("============= API Info =============")
	g.ApiLog.Infof("API access address: %s:%d", g.api.ip, g.api.port)
	g.ApiLog.Infoln("====================================")

	g.ApiLog.Infoln("API server started")
}

func (g *Gnb) stopApiServer() {
	g.ApiLog.Infoln("Stopping API server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := g.api.server.Shutdown(shutdownCtx); err != nil {
		g.ApiLog.Errorf("Failed to stop API server: %v", err)
	} else {
		g.ApiLog.Infoln("API server stopped successfully")
	}
}

func (g *Gnb) initApiRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Console GNB Info",
			Method:      http.MethodGet,
			Pattern:     "/info",
			HandlerFunc: g.handleConsoleGnbInfo,
		},
		{
			Name:        "Console GNB UE NRDC Modify",
			Method:      http.MethodPost,
			Pattern:     "/ue/nrdc",
			HandlerFunc: g.handleConsoleGnbUeNrdcModify,
		},
	}
}

func (g *Gnb) handleConsoleGnbInfo(c *gin.Context) {
	g.ApiLog.Infoln("Handling console get gnb info")

	plmnId := util.PlmnIdToModels(g.plmnId)
	snssai := util.SNssaiToModels(g.snssai)

	ranUeList := []consoleModel.RanUeInfo{}
	g.ranUeConns.Range(func(key, value any) bool {
		ranUe := key.(*RanUe)
		ranUeList = append(ranUeList, consoleModel.RanUeInfo{
			Imsi:          ranUe.GetMobileIdentityIMSI(),
			NrdcIndicator: ranUe.IsNrdcActivated(),
		})
		return true
	})

	xnUeList := []consoleModel.XnUeInfo{}
	g.xnUeConns.Range(func(key, value any) bool {
		xnUe := key.(*XnUe)
		xnUeList = append(xnUeList, consoleModel.XnUeInfo{
			Imsi: xnUe.GetIMSI(),
		})
		return true
	})

	c.JSON(http.StatusOK, consoleModel.ConsoleGnbInfoResponse{
		Message: "Get gNB info successful",
		GnbInfo: consoleModel.GnbInfo{
			GnbId:   hex.EncodeToString(g.gnbId),
			GnbName: g.gnbName,

			PlmnId: plmnId.Mcc + plmnId.Mnc,

			Snssai: consoleModel.SnssaiIE{
				Sst: strconv.Itoa(int(snssai.Sst)),
				Sd:  snssai.Sd,
			},

			RanUeList: ranUeList,
			XnUeList:  xnUeList,
		},
	})

	g.ApiLog.Infoln("Console get gnb info successful")
}

func (g *Gnb) handleConsoleGnbUeNrdcModify(c *gin.Context) {
	g.ApiLog.Infoln("Handling console gnb ue nrdc modify")

	var request consoleModel.ConsoleGnbUeNrdcModifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		g.ApiLog.Warnf("Error bind console gnb ue nrdc modify request: %v", err)
		c.JSON(http.StatusBadRequest, consoleModel.ConsoleGnbUeNrdcModifyResponse{
			Message: fmt.Sprintf("Error bind console gnb ue nrdc modify request: %v", err),
		})
		return
	}

	var ranUe *RanUe
	g.ranUeConns.Range(func(key, value any) bool {
		if key.(*RanUe).GetMobileIdentityIMSI() == request.Imsi {
			ranUe = key.(*RanUe)
		}
		return true
	})

	if ranUe == nil {
		g.ApiLog.Warnf("UE %s not found", request.Imsi)
		c.JSON(http.StatusNotFound, consoleModel.ConsoleGnbUeNrdcModifyResponse{
			Message: fmt.Sprintf("UE %s not found", request.Imsi),
		})
		return
	}
	if err := g.processUePduSessionModifyIndication(ranUe); err != nil {
		g.ApiLog.Errorf("Error process ue pdu session modify indication: %v", err)
		c.JSON(http.StatusInternalServerError, consoleModel.ConsoleGnbUeNrdcModifyResponse{
			Message: fmt.Sprintf("Error process ue pdu session modify indication: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, consoleModel.ConsoleGnbUeNrdcModifyResponse{
		Message: fmt.Sprintf("UE %s NRDC modify success", request.Imsi),
	})

	g.ApiLog.Infof("Console gnb ue %s nrdc control completed", request.Imsi)
}
