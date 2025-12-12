package gnb

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/free5gc/aper"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
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

	ngapPdu, err := ngap.Decoder(xnPdu.Data)
	if err != nil {
		g.XnLog.Warnf("Error decoding NGAP PDU: %v", err)
		return
	}

	switch ngapPdu.Present {
	case ngapType.NGAPPDUPresentInitiatingMessage:
		xnPduPresentInitiatingMessageDispatcher(g, conn, xnPdu.Imsi, ngapPdu)
	case ngapType.NGAPPDUPresentSuccessfulOutcome:
		xnPduPresentSuccessfulOutcomeDispatcher(g, conn, xnPdu.Imsi, ngapPdu)
	default:
		g.XnLog.Warnf("Unknown NGAP PDU Present: %v, expected %v or %v", ngapPdu.Present, ngapType.NGAPPDUPresentInitiatingMessage, ngapType.NGAPPDUPresentSuccessfulOutcome)
		return
	}
}

func xnPduPresentInitiatingMessageDispatcher(g *Gnb, conn net.Conn, imsi string, ngapPdu *ngapType.NGAPPDU) {
	switch ngapPdu.InitiatingMessage.ProcedureCode.Value {
	case ngapType.ProcedureCodePDUSessionResourceSetup:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Setup Request")
		xnPduSessionResourceSetupProcessor(g, conn, imsi, ngapPdu)
	case ngapType.ProcedureCodePDUSessionResourceModifyIndication:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Modify Indication")
		xnPduSessionResourceModifyIndicationProcessor(g, conn, imsi, ngapPdu)
	default:
		g.XnLog.Warnf("Unknown NGAP PDU Procedure Code: %v", ngapPdu.InitiatingMessage.ProcedureCode.Value)
		return
	}
}

func xnPduPresentSuccessfulOutcomeDispatcher(g *Gnb, conn net.Conn, imsi string, ngapPdu *ngapType.NGAPPDU) {
	switch ngapPdu.SuccessfulOutcome.ProcedureCode.Value {
	case ngapType.ProcedureCodePDUSessionResourceModifyIndication:
		g.XnLog.Infoln("Processing NGAP PDU Session Resource Modify Confirm")
		xnPduSessionResourceModifyConfirmProcessor(g, conn, imsi, ngapPdu)
	default:
		g.XnLog.Warnf("Unknown NGAP PDU Procedure Code: %v", ngapPdu.SuccessfulOutcome.ProcedureCode.Value)
		return
	}
}

func xnPduSessionResourceSetupProcessor(g *Gnb, conn net.Conn, imsi string, ngapPduSessionResourceSetup *ngapType.NGAPPDU) {
	var pduSessionResourceSetupRequestTransfer ngapType.PDUSessionResourceSetupRequestTransfer

	for _, ie := range ngapPduSessionResourceSetup.InitiatingMessage.Value.PDUSessionResourceSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq:
			for _, pduSessionResourceSetupItem := range ie.Value.PDUSessionResourceSetupListSUReq.List {
				if err := aper.UnmarshalWithParams(pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer, &pduSessionResourceSetupRequestTransfer, "valueExt"); err != nil {
					g.XnLog.Warnf("Error unmarshal pdu session resource setup request transfer: %v", err)
					return
				}
				g.XnLog.Tracef("Get PDUSessionResourceSetupRequestTransfer: %+v", pduSessionResourceSetupRequestTransfer)
			}
		case ngapType.ProtocolIEIDUEAggregateMaximumBitRate:
		}
	}

	xnUe := NewXnUe(imsi, g.teidGenerator.AllocateTeid(), nil)
	g.xnUeConns.Store(xnUe, struct{}{})
	g.XnLog.Debugf("Allocated DLTEID for XnUe: %s", hex.EncodeToString(xnUe.GetDlTeid()))

	for _, ie := range pduSessionResourceSetupRequestTransfer.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDPDUSessionAggregateMaximumBitRate:
		case ngapType.ProtocolIEIDULNGUUPTNLInformation:
		case ngapType.ProtocolIEIDAdditionalULNGUUPTNLInformation:
			xnUe.SetUlTeid(ie.Value.AdditionalULNGUUPTNLInformation.List[0].NGUUPTNLInformation.GTPTunnel.GTPTEID.Value)
		case ngapType.ProtocolIEIDPDUSessionType:
		case ngapType.ProtocolIEIDQosFlowSetupRequestList:
		}
	}

	// DC QoS Flow per TNL Information
	dcQosFlowPerTNLInformationItem := ngapType.QosFlowPerTNLInformationItem{}
	dcQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel

	// DC Transport Layer Information in QoS Flow per TNL Information
	dcUpTransportLayerInformation := &dcQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation
	dcUpTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	dcUpTransportLayerInformation.GTPTunnel = new(ngapType.GTPTunnel)
	dcUpTransportLayerInformation.GTPTunnel.GTPTEID.Value = xnUe.GetDlTeid()
	dcUpTransportLayerInformation.GTPTunnel.TransportLayerAddress = ngapConvert.IPAddressToNgap(g.ranN3Ip, "")

	// DC Associated QoS Flow List in QoS Flow per TNL Information
	dcAssociatedQosFlowList := &dcQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.AssociatedQosFlowList
	dcAssociatedQosFlowItem := ngapType.AssociatedQosFlowItem{}
	dcAssociatedQosFlowItem.QosFlowIdentifier.Value = 1
	dcAssociatedQosFlowList.List = append(dcAssociatedQosFlowList.List, dcAssociatedQosFlowItem)

	dcQosFlowPerTNLInformationMarshal, err := aper.MarshalWithParams(dcQosFlowPerTNLInformationItem, "valueExt")
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

func xnPduSessionResourceModifyIndicationProcessor(g *Gnb, conn net.Conn, imsi string, ngapPduSessionResourceModifyIndication *ngapType.NGAPPDU) {
	if xnReleaseUeProcessor(g, conn, imsi, ngapPduSessionResourceModifyIndication) {
		g.XnLog.Infof("XnUe released for imsi: %s", imsi)
		ngapPdu, err := ngap.Encoder(*ngapPduSessionResourceModifyIndication)
		if err != nil {
			g.XnLog.Warnf("Error encode ngap pdu: %v", err)
			return
		}
		g.XnLog.Tracef("Get NGAP PDU: %+v", ngapPdu)
		g.XnLog.Debugln("Get NGAP PDU")

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

		return
	}

	initiatingMessage := ngapPduSessionResourceModifyIndication.InitiatingMessage
	indication := initiatingMessage.Value.PDUSessionResourceModifyIndication

	var pduSessionResourceModifyIndicationIE *ngapType.PDUSessionResourceModifyIndicationIEs

	for _, ie := range indication.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDPDUSessionResourceModifyListModInd:
			pduSessionResourceModifyIndicationIE = &ie
		}
	}

	var pduSessionResourceModifyIndicationTransferMessageRaw []byte

	for _, pduSessionResourceModifyItem := range pduSessionResourceModifyIndicationIE.Value.PDUSessionResourceModifyListModInd.List {
		switch pduSessionResourceModifyItem.PDUSessionID.Value {
		case 4:
			pduSessionResourceModifyIndicationTransferMessageRaw = pduSessionResourceModifyItem.PDUSessionResourceModifyIndicationTransfer
		}
	}

	pduSessionResourceModifyIndicationTransfer := ngapType.PDUSessionResourceModifyIndicationTransfer{}
	if err := aper.UnmarshalWithParams(pduSessionResourceModifyIndicationTransferMessageRaw, &pduSessionResourceModifyIndicationTransfer, "valueExt"); err != nil {
		g.XnLog.Warnf("Error unmarshal pdu session resource modify indication transfer: %v", err)
		return
	}
	g.XnLog.Tracef("Get PDUSessionResourceModifyIndicationTransfer: %+v", pduSessionResourceModifyIndicationTransfer)

	xnUe := NewXnUe(imsi, g.teidGenerator.AllocateTeid(), nil)
	g.xnUeConns.Store(xnUe, struct{}{})
	g.XnLog.Debugf("Allocated DLTEID for XnUe: %s", hex.EncodeToString(xnUe.GetDlTeid()))

	// DC QoS Flow per TNL Information
	DCQosFlowPerTNLInformationItem := ngapType.QosFlowPerTNLInformationItem{}
	DCQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel

	// DC Transport Layer Information in QoS Flow per TNL Information
	DCUpTransportLayerInformation := &DCQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation
	DCUpTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	DCUpTransportLayerInformation.GTPTunnel = new(ngapType.GTPTunnel)
	DCUpTransportLayerInformation.GTPTunnel.GTPTEID.Value = xnUe.GetDlTeid()
	DCUpTransportLayerInformation.GTPTunnel.TransportLayerAddress = ngapConvert.IPAddressToNgap(g.ranN3Ip, "")

	// DC Associated QoS Flow List in QoS Flow per TNL Information
	DCAssociatedQosFlowList := &DCQosFlowPerTNLInformationItem.QosFlowPerTNLInformation.AssociatedQosFlowList
	DCAssociatedQosFlowItem := ngapType.AssociatedQosFlowItem{}
	DCAssociatedQosFlowItem.QosFlowIdentifier.Value = 1
	DCAssociatedQosFlowList.List = append(DCAssociatedQosFlowList.List, DCAssociatedQosFlowItem)

	// Additional DL QoS Flow per TNL Information
	pduSessionResourceModifyIndicationTransfer.AdditionalDLQosFlowPerTNLInformation = new(ngapType.QosFlowPerTNLInformationList)
	pduSessionResourceModifyIndicationTransfer.AdditionalDLQosFlowPerTNLInformation.List = append(pduSessionResourceModifyIndicationTransfer.AdditionalDLQosFlowPerTNLInformation.List, DCQosFlowPerTNLInformationItem)

	pduSessionResourceModifyIndicationTransferMarshal, err := aper.MarshalWithParams(pduSessionResourceModifyIndicationTransfer, "valueExt")
	if err != nil {
		g.XnLog.Warnf("Error marshal pdu session resource modify indication transfer: %v", err)
		return
	}

	for i := range pduSessionResourceModifyIndicationIE.Value.PDUSessionResourceModifyListModInd.List {
		switch pduSessionResourceModifyIndicationIE.Value.PDUSessionResourceModifyListModInd.List[i].PDUSessionID.Value {
		case 4:
			pduSessionResourceModifyIndicationIE.Value.PDUSessionResourceModifyListModInd.List[i].PDUSessionResourceModifyIndicationTransfer = pduSessionResourceModifyIndicationTransferMarshal
		}
	}
	g.XnLog.Tracef("Get PDUSessionResourceModifyIndicationTransfer: %+v", pduSessionResourceModifyIndicationTransfer)

	ngapPdu, err := ngap.Encoder(*ngapPduSessionResourceModifyIndication)
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

func xnPduSessionResourceModifyConfirmProcessor(g *Gnb, conn net.Conn, imsi string, ngapPduSessionResourceModifyConfirm *ngapType.NGAPPDU) {
	var pduSessionResourceModifyListModCfm *ngapType.PDUSessionResourceModifyListModCfm
	var pduSessionResourceModifyConfirmtransferRaw aper.OctetString

	for _, ie := range ngapPduSessionResourceModifyConfirm.SuccessfulOutcome.Value.PDUSessionResourceModifyConfirm.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
		case ngapType.ProtocolIEIDRANUENGAPID:
		case ngapType.ProtocolIEIDPDUSessionResourceModifyListModCfm:
			pduSessionResourceModifyListModCfm = ie.Value.PDUSessionResourceModifyListModCfm
		}
	}

	for _, pduSessionResourceModifyItem := range pduSessionResourceModifyListModCfm.List {
		switch pduSessionResourceModifyItem.PDUSessionID.Value {
		case 4:
			pduSessionResourceModifyConfirmtransferRaw = pduSessionResourceModifyItem.PDUSessionResourceModifyConfirmTransfer
		}
	}

	pduSessionResourceModifyConfirmtransfer := ngapType.PDUSessionResourceModifyConfirmTransfer{}
	if err := aper.UnmarshalWithParams(pduSessionResourceModifyConfirmtransferRaw, &pduSessionResourceModifyConfirmtransfer, "valueExt"); err != nil {
		g.XnLog.Warnf("Error unmarshal pdu session resource modify confirm transfer: %v", err)
		return
	}
	g.XnLog.Tracef("Get PDUSessionResourceModifyConfirmTransfer: %+v", pduSessionResourceModifyConfirmtransfer)

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

	xnUe.SetUlTeid(pduSessionResourceModifyConfirmtransfer.ULNGUUPTNLInformation.GTPTunnel.GTPTEID.Value)

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

func xnReleaseUeProcessor(g *Gnb, conn net.Conn, imsi string, ngapPduSessionResourceModifyConfirm *ngapType.NGAPPDU) bool {
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
