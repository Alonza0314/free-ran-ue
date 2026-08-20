package gnb

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/free-ran-ue/free-ran-ue/v2/constant"
	"github.com/free5gc/ngap/aper"
	"github.com/free5gc/ngap/ie"
	"github.com/free5gc/ngap/message"
)

type XnPdu struct {
	ImsiLength uint16
	Imsi       string
	Data       []byte
}

func NewXnPdu(imsi string, data []byte) *XnPdu {
	return &XnPdu{
		ImsiLength: 0,
		Imsi:       imsi,
		Data:       data,
	}
}

func (x *XnPdu) Marshal() ([]byte, error) {
	imsiBytes := []byte(x.Imsi)

	buffer := make([]byte, 2)
	binary.BigEndian.PutUint16(buffer, uint16(len(imsiBytes)))

	buffer = append(buffer, imsiBytes...)
	buffer = append(buffer, x.Data...)

	return buffer, nil
}

func (x *XnPdu) Unmarshal(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("data too short")
	}

	x.ImsiLength = binary.BigEndian.Uint16(data[:2])
	data = data[2:]

	if len(data) < int(x.ImsiLength) {
		return fmt.Errorf("data too short")
	}

	x.Imsi = string(data[:x.ImsiLength])
	data = data[x.ImsiLength:]

	x.Data = data

	return nil
}

func xnInterfaceProcessor(conn net.Conn, g *Gnb) {
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		g.XnLog.Warnf("Error reading XN packet: %v", err)
		return
	}
	g.XnLog.Tracef("Received %d bytes of XN packet: %+v", n, buffer[:n])
	g.XnLog.Debugln("Receive XN packet")

	xnPdu := XnPdu{}
	if err := xnPdu.Unmarshal(buffer[:n]); err != nil {
		g.XnLog.Errorf("Error unmarshal xn pdu: %v", err)
		return
	}
	g.XnLog.Tracef("Received XN PDU: %+v", xnPdu)
	g.XnLog.Debugln("Receive XN PDU")

	ngapMsg, err := message.Parse(xnPdu.Data)
	if err != nil {
		g.XnLog.Warnf("Error decoding NGAP PDU: %v", err)
		return
	}

	switch msg := ngapMsg.(type) {
	case *message.PDUSessionResourceSetupRequest:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Setup Request")
		xnPduSessionResourceSetupProcessor(g, conn, xnPdu.Imsi, msg)
	case *message.PDUSessionResourceModifyIndication:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Modify Indication")
		xnPduSessionResourceModifyIndicationProcessor(g, conn, xnPdu.Imsi, msg, xnPdu.Data)
	case *message.PDUSessionResourceModifyConfirm:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Modify Confirm")
		xnPduSessionResourceModifyConfirmProcessor(g, conn, xnPdu.Imsi, msg)
	default:
		g.XnLog.Warnf("Unknown NGAP PDU message: %T", ngapMsg)
	}
}

func buildDCQosFlowPerTNLInformationItem(dlTeid []byte, ranN3Ip string) ie.QosFlowPerTNLInformationItem {
	return ie.QosFlowPerTNLInformationItem{
		QosFlowPerTNLInformation: &ie.QosFlowPerTNLInformation{
			UPTransportLayerInformation: &ie.UPTransportLayerInformation{
				Choice: &ie.GTPTunnel{
					GTPTEID: &ie.GTPTEID{
						Value: dlTeid,
					},
					TransportLayerAddress: &ie.TransportLayerAddress{
						Value: ngapConvertIPAddressToNgap(ranN3Ip),
					},
				},
			},
			AssociatedQosFlowList: &ie.AssociatedQosFlowList{
				List: []ie.AssociatedQosFlowItem{
					{
						QosFlowIdentifier: &ie.QosFlowIdentifier{
							Value: 1,
						},
					},
				},
			},
		},
	}
}

func xnPduSessionResourceSetupProcessor(g *Gnb, conn net.Conn, imsi string, msg *message.PDUSessionResourceSetupRequest) {
	var pduSessionResourceSetupRequestTransfer ie.PDUSessionResourceSetupRequestTransfer

	if msg.PDUSessionResourceSetupListSUReq != nil {
		for _, pduSessionResourceSetupItem := range msg.PDUSessionResourceSetupListSUReq.List {
			if pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer == nil {
				continue
			}
			if err := ie.UnmarshalBinary(*pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer, &pduSessionResourceSetupRequestTransfer); err != nil {
				g.XnLog.Warnf("Error unmarshal pdu session resource setup request transfer: %v", err)
				return
			}
			g.XnLog.Tracef("Get PDUSessionResourceSetupRequestTransfer: %+v", pduSessionResourceSetupRequestTransfer)
		}
	}

	xnUe := NewXnUe(imsi, g.teidGenerator.AllocateTeid(), nil)
	g.xnUeConns.Store(xnUe, struct{}{})
	g.XnLog.Debugf("Allocated DLTEID for XnUe: %s", hex.EncodeToString(xnUe.GetDlTeid()))

	if pduSessionResourceSetupRequestTransfer.ProtocolIEs != nil {
		for _, item := range pduSessionResourceSetupRequestTransfer.ProtocolIEs.List {
			if item.AdditionalULNGUUPTNLInformation == nil || len(item.AdditionalULNGUUPTNLInformation.List) == 0 {
				continue
			}
			nguUpTnlInformation := item.AdditionalULNGUUPTNLInformation.List[0].NGUUPTNLInformation
			if nguUpTnlInformation == nil {
				continue
			}
			if gtpTunnel, ok := nguUpTnlInformation.Choice.(*ie.GTPTunnel); ok && gtpTunnel.GTPTEID != nil {
				xnUe.SetUlTeid(gtpTunnel.GTPTEID.Value)
			}
		}
	}

	// DC QoS Flow per TNL Information
	dcQosFlowPerTNLInformationItem := buildDCQosFlowPerTNLInformationItem(xnUe.GetDlTeid(), g.ranN3Ip)

	dcQosFlowPerTNLInformationMarshal, err := ie.MarshalBinary(&dcQosFlowPerTNLInformationItem)
	if err != nil {
		g.XnLog.Warnf("Error marshal dc qos flow per tnl information: %v", err)
		return
	}

	xnPdu := NewXnPdu(imsi, dcQosFlowPerTNLInformationMarshal)
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		g.XnLog.Warnf("Error marshal xn pdu: %v", err)
		return
	}

	n, err := conn.Write(xnPduBytes)
	if err != nil {
		g.XnLog.Warnf("Error write dc qos flow per tnl information: %v", err)
		return
	}
	g.XnLog.Tracef("Sent %d bytes of DC QoS Flow per TNL Information to XN", n)
	g.XnLog.Debugln("Send DC QoS Flow per TNL Information to XN")

	g.dlTeidToUe.Store(hex.EncodeToString(xnUe.GetDlTeid()), xnUe)
	g.XnLog.Debugf("Stored XN UE %s with DL TEID %s to dlTeidToUe", xnUe.GetIMSI(), hex.EncodeToString(xnUe.GetDlTeid()))

	g.imsiTodlTeidAndUeType.Store(imsi, dlTeidAndUeType{
		dlTeid: xnUe.GetDlTeid(),
		ueType: constant.UE_TYPE_XN,
	})
	g.XnLog.Debugf("Sent DL TEID %s to imsiTodlTeidAndUeType", hex.EncodeToString(xnUe.GetDlTeid()))
}

func xnPduSessionResourceModifyIndicationProcessor(g *Gnb, conn net.Conn, imsi string, msg *message.PDUSessionResourceModifyIndication, ngapRaw []byte) {
	if xnReleaseUeProcessor(g, conn, imsi) {
		g.XnLog.Infof("XnUe released for imsi: %s", imsi)

		xnPdu := NewXnPdu(imsi, ngapRaw)
		xnPduBytes, err := xnPdu.Marshal()
		if err != nil {
			g.XnLog.Warnf("Error marshal xn pdu: %v", err)
			return
		}

		n, err := conn.Write(xnPduBytes)
		if err != nil {
			g.XnLog.Warnf("Error write ngap pdu: %v", err)
			return
		}
		g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Modify Indication to XN", n)
		g.XnLog.Debugln("Send NGAP PDU Session Resource Modify Indication to XN")

		return
	}

	if msg.PDUSessionResourceModifyListModInd == nil {
		g.XnLog.Warnf("Error pdu session resource modify indication: missing mandatory IE")
		return
	}

	itemIndex := -1
	for i, item := range msg.PDUSessionResourceModifyListModInd.List {
		if item.PDUSessionID != nil && item.PDUSessionID.Value == constant.PDU_SESSION_ID {
			itemIndex = i
			break
		}
	}
	if itemIndex == -1 || msg.PDUSessionResourceModifyListModInd.List[itemIndex].PDUSessionResourceModifyIndicationTransfer == nil {
		g.XnLog.Warnf("Error pdu session resource modify indication: pdu session %d not found", constant.PDU_SESSION_ID)
		return
	}

	var pduSessionResourceModifyIndicationTransfer ie.PDUSessionResourceModifyIndicationTransfer
	if err := ie.UnmarshalBinary(*msg.PDUSessionResourceModifyListModInd.List[itemIndex].PDUSessionResourceModifyIndicationTransfer, &pduSessionResourceModifyIndicationTransfer); err != nil {
		g.XnLog.Warnf("Error unmarshal pdu session resource modify indication transfer: %v", err)
		return
	}
	g.XnLog.Tracef("Get PDUSessionResourceModifyIndicationTransfer: %+v", pduSessionResourceModifyIndicationTransfer)

	xnUe := NewXnUe(imsi, g.teidGenerator.AllocateTeid(), nil)
	g.xnUeConns.Store(xnUe, struct{}{})
	g.XnLog.Debugf("Allocated DLTEID for XnUe: %s", hex.EncodeToString(xnUe.GetDlTeid()))

	// DC QoS Flow per TNL Information
	dcQosFlowPerTNLInformationItem := buildDCQosFlowPerTNLInformationItem(xnUe.GetDlTeid(), g.ranN3Ip)

	// Additional DL QoS Flow per TNL Information
	pduSessionResourceModifyIndicationTransfer.AdditionalDLQosFlowPerTNLInformation = &ie.QosFlowPerTNLInformationList{
		List: []ie.QosFlowPerTNLInformationItem{dcQosFlowPerTNLInformationItem},
	}

	pduSessionResourceModifyIndicationTransferMarshal, err := ie.MarshalBinary(&pduSessionResourceModifyIndicationTransfer)
	if err != nil {
		g.XnLog.Warnf("Error marshal pdu session resource modify indication transfer: %v", err)
		return
	}

	transfer := aper.OctetString(pduSessionResourceModifyIndicationTransferMarshal)
	msg.PDUSessionResourceModifyListModInd.List[itemIndex].PDUSessionResourceModifyIndicationTransfer = &transfer
	g.XnLog.Tracef("Get PDUSessionResourceModifyIndicationTransfer: %+v", pduSessionResourceModifyIndicationTransfer)

	ngapPdu, err := msg.MarshalBinary()
	if err != nil {
		g.XnLog.Warnf("Error encode ngap pdu: %v", err)
		return
	}

	xnPdu := NewXnPdu(imsi, ngapPdu)
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		g.XnLog.Warnf("Error marshal xn pdu: %v", err)
		return
	}

	n, err := conn.Write(xnPduBytes)
	if err != nil {
		g.XnLog.Warnf("Error write ngap pdu: %v", err)
		return
	}
	g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Modify Indication to XN", n)
	g.XnLog.Debugln("Send NGAP PDU Session Resource Modify Indication to XN")
}

func xnPduSessionResourceModifyConfirmProcessor(g *Gnb, conn net.Conn, imsi string, msg *message.PDUSessionResourceModifyConfirm) {
	if msg.PDUSessionResourceModifyListModCfm == nil {
		g.XnLog.Warnf("Error pdu session resource modify confirm: missing mandatory IE")
		return
	}

	var pduSessionResourceModifyConfirmTransferRaw *aper.OctetString
	for _, pduSessionResourceModifyItem := range msg.PDUSessionResourceModifyListModCfm.List {
		if pduSessionResourceModifyItem.PDUSessionID != nil && pduSessionResourceModifyItem.PDUSessionID.Value == constant.PDU_SESSION_ID {
			pduSessionResourceModifyConfirmTransferRaw = pduSessionResourceModifyItem.PDUSessionResourceModifyConfirmTransfer
		}
	}
	if pduSessionResourceModifyConfirmTransferRaw == nil {
		g.XnLog.Warnf("Error pdu session resource modify confirm: pdu session %d not found", constant.PDU_SESSION_ID)
		return
	}

	pduSessionResourceModifyConfirmTransfer := ie.PDUSessionResourceModifyConfirmTransfer{}
	if err := ie.UnmarshalBinary(*pduSessionResourceModifyConfirmTransferRaw, &pduSessionResourceModifyConfirmTransfer); err != nil {
		g.XnLog.Warnf("Error unmarshal pdu session resource modify confirm transfer: %v", err)
		return
	}
	g.XnLog.Tracef("Get PDUSessionResourceModifyConfirmTransfer: %+v", pduSessionResourceModifyConfirmTransfer)

	var xnUe *XnUe

	g.xnUeConns.Range(func(key, value interface{}) bool {
		if key.(*XnUe).GetIMSI() == imsi {
			xnUe = key.(*XnUe)
			return false
		}
		return true
	})

	if xnUe == nil {
		g.XnLog.Warnf("XnUe not found for imsi: %s", imsi)
		return
	}

	if pduSessionResourceModifyConfirmTransfer.ULNGUUPTNLInformation != nil {
		if gtpTunnel, ok := pduSessionResourceModifyConfirmTransfer.ULNGUUPTNLInformation.Choice.(*ie.GTPTunnel); ok && gtpTunnel.GTPTEID != nil {
			xnUe.SetUlTeid(gtpTunnel.GTPTEID.Value)
		}
	}

	xnPdu := NewXnPdu(imsi, []byte{})
	xnPduBytes, err := xnPdu.Marshal()
	if err != nil {
		g.XnLog.Warnf("Error marshal xn pdu: %v", err)
		return
	}

	n, err := conn.Write(xnPduBytes)
	if err != nil {
		g.XnLog.Warnf("Error write ngap pdu: %v", err)
		return
	}
	g.XnLog.Tracef("Sent %d bytes of NGAP PDU Session Resource Modify Confirm to XN", n)
	g.XnLog.Debugln("Send NGAP PDU Session Resource Modify Confirm to XN")

	g.dlTeidToUe.Store(hex.EncodeToString(xnUe.GetDlTeid()), xnUe)
	g.XnLog.Debugf("Stored XN UE %s with DL TEID %s to dlTeidToUe", xnUe.GetIMSI(), hex.EncodeToString(xnUe.GetDlTeid()))

	g.imsiTodlTeidAndUeType.Store(imsi, dlTeidAndUeType{
		dlTeid: xnUe.GetDlTeid(),
		ueType: constant.UE_TYPE_XN,
	})
	g.XnLog.Debugf("Sent DL TEID %s to imsiTodlTeidAndUeType", hex.EncodeToString(xnUe.GetDlTeid()))
}

func xnReleaseUeProcessor(g *Gnb, conn net.Conn, imsi string) bool {
	var xnUe *XnUe

	g.xnUeConns.Range(func(key, value interface{}) bool {
		if key.(*XnUe).GetIMSI() == imsi {
			xnUe = key.(*XnUe)
			return false
		}
		return true
	})

	if xnUe == nil {
		return false
	}

	g.dlTeidToUe.Delete(hex.EncodeToString(xnUe.GetDlTeid()))
	g.XnLog.Debugf("Deleted XN UE %s with DL TEID %s from dlTeidToUe", xnUe.GetIMSI(), hex.EncodeToString(xnUe.GetDlTeid()))

	g.addressToUe.Delete(xnUe.GetDataPlaneAddress().String())
	g.XnLog.Debugf("Deleted XN UE %s with data plane address %s from addressToUe", xnUe.GetIMSI(), xnUe.GetDataPlaneAddress().String())

	xnUe.Release(g.teidGenerator)
	g.XnLog.Debugf("Released XN UE %s with DL TEID %s", xnUe.GetIMSI(), hex.EncodeToString(xnUe.GetDlTeid()))

	g.xnUeConns.Delete(xnUe)
	g.XnLog.Debugf("Deleted XN UE %s from xnUeConns", xnUe.GetIMSI())

	return true
}
