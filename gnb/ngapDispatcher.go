package gnb

import (
	"errors"
	"net"

	"github.com/Alonza0314/free-ran-ue/constant"
	"github.com/free5gc/aper"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
)

type ngapDispatcher struct{}

func (d *ngapDispatcher) start(g *Gnb) {
	g.NgapLog.Infoln("NGAP dispatcher started")
	ngapBuffer := make([]byte, 1024)
	for {
		n, err := g.n2Conn.Read(ngapBuffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				g.NgapLog.Debugln("NGAP dispatcher closed")
				return
			}
			g.NgapLog.Errorf("Error reading NGAP buffer: %v", err)
			continue
		}
		g.NgapLog.Tracef("Received %d bytes of NGAP packet: %+v", n, ngapBuffer[:n])
		g.NgapLog.Debugln("Receive NGAP packet")

		tmp := make([]byte, n)
		copy(tmp, ngapBuffer[:n])
		go d.dispatch(g, tmp)
	}
}

func (d *ngapDispatcher) dispatch(g *Gnb, ngapRaw []byte) {
	ngapPdu, err := ngap.Decoder(ngapRaw)
	if err != nil {
		g.NgapLog.Errorf("Error decoding NGAP PDU: %v", err)
		return
	}

	switch ngapPdu.Present {
	case ngapType.NGAPPDUPresentInitiatingMessage:
		d.initiatingMessageProcessor(g, ngapPdu, ngapRaw)
	case ngapType.NGAPPDUPresentSuccessfulOutcome:
		d.successfulOutcomeProcessor(g, ngapPdu, ngapRaw)
	default:
		g.NgapLog.Warnf("Unknown NGAP PDU Present: %v", ngapPdu.Present)
		return
	}
}

func (d *ngapDispatcher) initiatingMessageProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU, ngapRaw []byte) {
	switch ngapPdu.InitiatingMessage.ProcedureCode.Value {
	case ngapType.ProcedureCodeDownlinkNASTransport:
		g.NgapLog.Debugln("Processing NGAP Downlink NAS Transport")
		d.downLinkNASTransportProcessor(g, ngapPdu)
	case ngapType.ProcedureCodeInitialContextSetup:
		g.NgapLog.Debugln("Processing NGAP Initial Context Setup")
		d.initialContextSetupProcessor(g, ngapPdu)
	case ngapType.ProcedureCodePDUSessionResourceSetup:
		g.NgapLog.Debugln("Processing NGAP PDU Session Resource Setup")
		d.pduSessionResourceSetupProcessor(g, ngapPdu, ngapRaw)
	case ngapType.ProcedureCodeUEContextRelease:
		g.NgapLog.Debugln("Processing NGAP UE Context Release")
		d.ueContextReleaseProcessor(g, ngapPdu)
	default:
		g.NgapLog.Warnf("Unknown NGAP PDU Initiating Message Procedure Code: %v", ngapPdu.InitiatingMessage.ProcedureCode.Value)
	}
}

func (d *ngapDispatcher) successfulOutcomeProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU, ngapRaw []byte) {
	switch ngapPdu.SuccessfulOutcome.ProcedureCode.Value {
	case ngapType.ProcedureCodePDUSessionResourceModifyIndication:
		g.NgapLog.Debugln("Processing NGAP PDU Session Resource Modify Indication")
		d.pduSessionResourceModifyIndicationProcessor(g, ngapPdu, ngapRaw)
	default:
		g.NgapLog.Warnf("Unknown NGAP PDU Successful Outcome Procedure Code: %v", ngapPdu.SuccessfulOutcome.ProcedureCode.Value)
	}
}

func (d *ngapDispatcher) downLinkNASTransportProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU) {
	var (
		downLinkNASTransportMessage []byte
		amfUeNgapId                 int64
		ranUeNgapId                 int64
	)

	for _, ie := range ngapPdu.InitiatingMessage.Value.DownlinkNASTransport.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID.Value
		case ngapType.ProtocolIEIDRANUENGAPID:
			ranUeNgapId = ie.Value.RANUENGAPID.Value
		case ngapType.ProtocolIEIDNASPDU:
			if ie.Value.NASPDU == nil {
				g.NgapLog.Errorf("Error downlink NAS transport: NASPDU is nil")
				return
			}
			downLinkNASTransportMessage = make([]byte, len(ie.Value.NASPDU.Value))
			copy(downLinkNASTransportMessage, ie.Value.NASPDU.Value)
			g.NgapLog.Tracef("Get downlink NAS transport message: %+v", downLinkNASTransportMessage)
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error downlink NAS transport: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

	if ranUe.GetAmfUeId() == -1 {
		ranUe.SetAmfUeId(amfUeNgapId)
	}

	n, err := ranUe.GetN1Conn().Write(downLinkNASTransportMessage)
	if err != nil {
		g.NgapLog.Errorf("Error send downlink NAS transport message to UE: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of downlink NAS transport message to UE", n)
	g.NgapLog.Debugf("Send downlink NAS transport message to UE %s", ranUe.GetMobileIdentityIMSI())
}

func (d *ngapDispatcher) initialContextSetupProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU) {
	var (
		nasPdu      []byte
		amfUeNgapId int64
		ranUeNgapId int64
	)

	for _, ie := range ngapPdu.InitiatingMessage.Value.InitialContextSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID.Value
		case ngapType.ProtocolIEIDRANUENGAPID:
			ranUeNgapId = ie.Value.RANUENGAPID.Value
		case ngapType.ProtocolIEIDNASPDU:
			if ie.Value.NASPDU == nil {
				g.NgapLog.Errorf("Error initial context setup: NASPDU is nil")
				return
			}
			nasPdu = make([]byte, len(ie.Value.NASPDU.Value))
			copy(nasPdu, ie.Value.NASPDU.Value)
			g.NgapLog.Tracef("Get initial context setup NASPDU: %+v", nasPdu)
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error initial context setup: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

	if ranUe.GetAmfUeId() != amfUeNgapId {
		g.NgapLog.Errorf("Error initial context setup: Ran UE with ranUeNgapId %d has amfUeNgapId %d, expected %d", ranUeNgapId, ranUe.GetAmfUeId(), amfUeNgapId)
		return
	}

	initialContextSetupResponse, err := getNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId)
	if err != nil {
		g.NgapLog.Errorf("Error get initial context setup response: %v", err)
		return
	}
	g.NgapLog.Tracef("Get initial context setup response: %+v", initialContextSetupResponse)

	n, err := g.n2Conn.Write(initialContextSetupResponse)
	if err != nil {
		g.NgapLog.Errorf("Error send initial context setup response to AMF: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of initial context setup response to AMF", n)
	g.NgapLog.Debugln("Send initial context setup response to AMF")

	n, err = ranUe.GetN1Conn().Write(nasPdu)
	if err != nil {
		g.NgapLog.Errorf("Error send initial context setup NASPDU to UE: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of initial context setup NASPDU to UE", n)
	g.NgapLog.Debugln("Send initial context setup NASPDU to UE %s", ranUe.GetMobileIdentityIMSI())
}

func (d *ngapDispatcher) pduSessionResourceSetupProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU, ngapRaw []byte) {
	var (
		nasPdu      []byte
		amfUeNgapId int64
		ranUeNgapId int64
		err         error

		pduSessionResourceSetupRequestTransfer *ngapType.PDUSessionResourceSetupRequestTransfer
	)

	for _, ie := range ngapPdu.InitiatingMessage.Value.PDUSessionResourceSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID.Value
		case ngapType.ProtocolIEIDRANUENGAPID:
			ranUeNgapId = ie.Value.RANUENGAPID.Value
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq:
			for _, pduSessionResourceSetupItem := range ie.Value.PDUSessionResourceSetupListSUReq.List {
				nasPdu = make([]byte, len(pduSessionResourceSetupItem.PDUSessionNASPDU.Value))
				copy(nasPdu, pduSessionResourceSetupItem.PDUSessionNASPDU.Value)
				g.NgapLog.Tracef("Get PDU Session Resource Setup NASPDU: %+v", nasPdu)

				if err := aper.UnmarshalWithParams(pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer, &pduSessionResourceSetupRequestTransfer, "valueExt"); err != nil {
					g.NgapLog.Errorf("Error unmarshal pdu session resource setup request transfer: %v", err)
					return
				}
				g.NgapLog.Tracef("Get PDU Session Resource Setup Request Transfer: %+v", pduSessionResourceSetupRequestTransfer)
			}
		case ngapType.ProtocolIEIDUEAggregateMaximumBitRate:
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error pdu session resource setup: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

	if ranUe.GetAmfUeId() != amfUeNgapId {
		g.NgapLog.Errorf("Error pdu session resource setup: Ran UE with ranUeNgapId %d has amfUeNgapId %d, expected %d", ranUeNgapId, ranUe.GetAmfUeId(), amfUeNgapId)
		return
	}

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

	var qosFlowPerTNLInformationItem ngapType.QosFlowPerTNLInformationItem
	if ranUe.IsNrdcActivated() {
		if qosFlowPerTNLInformationItem, err = g.xnPduSessionResourceSetupRequestTransfer(ranUe.GetMobileIdentityIMSI(), ngapRaw); err != nil {
			g.XnLog.Warnf("Error xn pdu session resource setup request transfer: %v", err)
		}
	}

	n, err := ranUe.GetN1Conn().Write(nasPdu)
	if err != nil {
		g.NgapLog.Errorf("Error send pdu session resource setup NASPDU to UE: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of pdu session resource setup NASPDU to UE", n)
	g.NgapLog.Debugln("Send pdu session resource setup NASPDU to UE")

	ngapPduSessionResourceSetupResponseTransfer, err := getPduSessionResourceSetupResponseTransfer(ranUe.GetDlTeid(), g.ranN3Ip, 1, g.staticNrdc, qosFlowPerTNLInformationItem)
	if err != nil {
		g.NgapLog.Errorf("Error get pdu session resource setup response transfer: %v", err)
		return
	}
	g.NgapLog.Tracef("Get pdu session resource setup response transfer: %+v", ngapPduSessionResourceSetupResponseTransfer)

	ngapPduSessionResourceSetupResponse, err := getPduSessionResourceSetupResponse(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), constant.PDU_SESSION_ID, ngapPduSessionResourceSetupResponseTransfer)
	if err != nil {
		g.NgapLog.Errorf("Error get pdu session resource setup response: %v", err)
		return
	}
	g.NgapLog.Tracef("Get pdu session resource setup response: %+v", ngapPduSessionResourceSetupResponse)

	n, err = g.n2Conn.Write(ngapPduSessionResourceSetupResponse)
	if err != nil {
		g.NgapLog.Errorf("Error send pdu session resource setup response to AMF: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of pdu session resource setup response to AMF", n)
	g.NgapLog.Debugln("Send pdu session resource setup response to AMF")

	ranUe.GetPduSessionEstablishmentCompleteChan() <- struct{}{}
}

func (d *ngapDispatcher) ueContextReleaseProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU) {
	var (
		amfUeNgapId int64
		ranUeNgapId int64
	)

	for _, ie := range ngapPdu.InitiatingMessage.Value.UEContextReleaseCommand.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDUENGAPIDs:
			amfUeNgapId, ranUeNgapId = ie.Value.UENGAPIDs.UENGAPIDPair.AMFUENGAPID.Value, ie.Value.UENGAPIDs.UENGAPIDPair.RANUENGAPID.Value
		case ngapType.ProtocolIEIDCause:
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error ue context release: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

	if ranUe.GetAmfUeId() != amfUeNgapId {
		g.NgapLog.Errorf("Error ue context release: Ran UE with ranUeNgapId %d has amfUeNgapId %d, expected %d", ranUeNgapId, ranUe.GetAmfUeId(), amfUeNgapId)
		return
	}

	ngapUeContextReleaseCompleteMessage, err := getNgapUeContextReleaseCompleteMessage(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), []int64{constant.PDU_SESSION_ID}, g.plmnId, g.tai)
	if err != nil {
		g.NgapLog.Errorf("Error get ngap ue context release complete message: %v", err)
		return
	}
	g.NgapLog.Tracef("Get ngap ue context release complete message: %+v", ngapUeContextReleaseCompleteMessage)

	n, err := g.n2Conn.Write(ngapUeContextReleaseCompleteMessage)
	if err != nil {
		g.NgapLog.Errorf("Error send ngap ue context release complete message to AMF: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of ngap ue context release complete message to AMF", n)
	g.NgapLog.Debugln("Send ngap ue context release complete message to AMF")

	ranUe.GetUeContextReleaseCompleteChan() <- struct{}{}
}

func (d *ngapDispatcher) pduSessionResourceModifyIndicationProcessor(g *Gnb, ngapPdu *ngapType.NGAPPDU, ngapRaw []byte) {
	var (
		amfUeNgapId int64
		ranUeNgapId int64
	)

	for _, ie := range ngapPdu.SuccessfulOutcome.Value.PDUSessionResourceModifyConfirm.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID.Value
		case ngapType.ProtocolIEIDRANUENGAPID:
			ranUeNgapId = ie.Value.RANUENGAPID.Value
		case ngapType.ProtocolIEIDPDUSessionResourceModifyListModCfm:
			g.NgapLog.Infof("ran ue with ranUeNgapId %d pdu session resource modify indication successful", ranUeNgapId)
		case ngapType.ProtocolIEIDPDUSessionResourceFailedToModifyListModCfm:
			g.NgapLog.Errorf("ran ue with ranUeNgapId %d pdu session resource modify indication failed", ranUeNgapId)
			return
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error pdu session resource modify indication: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

	if ranUe.GetAmfUeId() != amfUeNgapId {
		g.NgapLog.Errorf("Error pdu session resource modify indication: Ran UE with ranUeNgapId %d has amfUeNgapId %d, expected %d", ranUeNgapId, ranUe.GetAmfUeId(), amfUeNgapId)
		return
	}

	// send confirm to Xm for update xnUE ULTEID
	if !ranUe.IsNrdcActivated() {
		if _, err := g.xnPduSessionResourceModifyConfirm(ranUe.GetMobileIdentityIMSI(), ngapRaw); err != nil {
			g.XnLog.Errorf("Error xn pdu session resource modify confirm: %v", err)
			return
		}
		g.XnLog.Debugln("XN PDU Session Resource Modify Confirm sent")
	}

	// send modify message to UE
	modifyMessage := []byte(constant.UE_TUNNEL_UPDATE)

	n, err := ranUe.GetN1Conn().Write(modifyMessage)
	if err != nil {
		g.NgapLog.Errorf("Error send modify message to UE: %v", err)
		return
	}
	g.NgapLog.Tracef("Sent %d bytes of modify message to UE", n)
	g.NgapLog.Debugln("Send Modify Message to UE")

	// update ranUe NRDC status
	if ranUe.IsNrdcActivated() {
		ranUe.DeactivateNrdc()
		g.NgapLog.Infof("UE %s NRDC deactivated", ranUe.GetMobileIdentityIMSI())
	} else {
		ranUe.ActivateNrdc()
		g.NgapLog.Infof("UE %s NRDC activated", ranUe.GetMobileIdentityIMSI())
	}

	ranUe.GetPduSessionModifyIndicationCompleteChan() <- struct{}{}
}
