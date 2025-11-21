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
		ranUeNgapId int64
		err         error

		pduSessionResourceSetupRequestTransfer *ngapType.PDUSessionResourceSetupRequestTransfer
	)

	for _, ie := range ngapPdu.InitiatingMessage.Value.PDUSessionResourceSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
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
	var ranUeNgapId int64

	for _, ie := range ngapPdu.InitiatingMessage.Value.UEContextReleaseCommand.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDUENGAPIDs:
			ranUeNgapId = ie.Value.UENGAPIDs.UENGAPIDPair.RANUENGAPID.Value
		case ngapType.ProtocolIEIDCause:
		}
	}

	ueValue, exist := g.ranUeConns.Load(ranUeNgapId)
	if !exist {
		g.NgapLog.Errorf("Error ue context release: Ran UE with ranUeNgapId %d not found", ranUeNgapId)
		return
	}
	ranUe := ueValue.(*RanUe)

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
