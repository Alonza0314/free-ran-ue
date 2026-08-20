package gnb

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/free-ran-ue/free-ran-ue/v2/constant"
	"github.com/free5gc/ngap/ie"
	"github.com/free5gc/ngap/message"
)

type ngapDispatcher struct{}

func (d *ngapDispatcher) start(g *Gnb) {
	g.NgapLog.Infoln("NGAP dispatcher started")
	ngapBuffer := make([]byte, 1024)
	for {
		n, err := g.n2Conn.Read(ngapBuffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) || strings.Contains(err.Error(), "bad file descriptor") {
				g.NgapLog.Debugln("NGAP connection closed")
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
	ngapMsg, err := message.Parse(ngapRaw)
	if err != nil {
		g.NgapLog.Errorf("Error decoding NGAP PDU: %v", err)
		return
	}

	switch msg := ngapMsg.(type) {
	case *message.DownlinkNASTransport:
		g.NgapLog.Debugln("Processing NGAP Downlink NAS Transport")
		d.downLinkNASTransportProcessor(g, msg)
	case *message.InitialContextSetupRequest:
		g.NgapLog.Debugln("Processing NGAP Initial Context Setup")
		d.initialContextSetupProcessor(g, msg)
	case *message.PDUSessionResourceSetupRequest:
		g.NgapLog.Debugln("Processing NGAP PDU Session Resource Setup")
		d.pduSessionResourceSetupProcessor(g, msg, ngapRaw)
	case *message.UEContextReleaseCommand:
		g.NgapLog.Debugln("Processing NGAP UE Context Release")
		d.ueContextReleaseProcessor(g, msg)
	case *message.PDUSessionResourceModifyConfirm:
		g.NgapLog.Debugln("Processing NGAP PDU Session Resource Modify Indication")
		d.pduSessionResourceModifyIndicationProcessor(g, msg, ngapRaw)
	default:
		g.NgapLog.Warnf("Unknown NGAP PDU message: %T", ngapMsg)
	}
}

func (d *ngapDispatcher) downLinkNASTransportProcessor(g *Gnb, msg *message.DownlinkNASTransport) {
	if msg.AMFUENGAPID == nil || msg.RANUENGAPID == nil || msg.NASPDU == nil {
		g.NgapLog.Errorf("Error downlink NAS transport: missing mandatory IE")
		return
	}

	amfUeNgapId := msg.AMFUENGAPID.Value
	ranUeNgapId := msg.RANUENGAPID.Value

	downLinkNASTransportMessage := make([]byte, len(msg.NASPDU.Value))
	copy(downLinkNASTransportMessage, msg.NASPDU.Value)
	g.NgapLog.Tracef("Get downlink NAS transport message: %+v", downLinkNASTransportMessage)

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

func (d *ngapDispatcher) initialContextSetupProcessor(g *Gnb, msg *message.InitialContextSetupRequest) {
	if msg.AMFUENGAPID == nil || msg.RANUENGAPID == nil || msg.NASPDU == nil {
		g.NgapLog.Errorf("Error initial context setup: missing mandatory IE")
		return
	}

	amfUeNgapId := msg.AMFUENGAPID.Value
	ranUeNgapId := msg.RANUENGAPID.Value

	nasPdu := make([]byte, len(msg.NASPDU.Value))
	copy(nasPdu, msg.NASPDU.Value)
	g.NgapLog.Tracef("Get initial context setup NASPDU: %+v", nasPdu)

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

func (d *ngapDispatcher) pduSessionResourceSetupProcessor(g *Gnb, msg *message.PDUSessionResourceSetupRequest, ngapRaw []byte) {
	if msg.AMFUENGAPID == nil || msg.RANUENGAPID == nil || msg.PDUSessionResourceSetupListSUReq == nil {
		g.NgapLog.Errorf("Error pdu session resource setup: missing mandatory IE")
		return
	}

	amfUeNgapId := msg.AMFUENGAPID.Value
	ranUeNgapId := msg.RANUENGAPID.Value

	var (
		nasPdu                                 []byte
		pduSessionResourceSetupRequestTransfer ie.PDUSessionResourceSetupRequestTransfer
	)

	for _, pduSessionResourceSetupItem := range msg.PDUSessionResourceSetupListSUReq.List {
		if pduSessionResourceSetupItem.PDUSessionNASPDU != nil {
			nasPdu = make([]byte, len(pduSessionResourceSetupItem.PDUSessionNASPDU.Value))
			copy(nasPdu, pduSessionResourceSetupItem.PDUSessionNASPDU.Value)
			g.NgapLog.Tracef("Get PDU Session Resource Setup NASPDU: %+v", nasPdu)
		}

		if pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer == nil {
			continue
		}
		if err := ie.UnmarshalBinary(*pduSessionResourceSetupItem.PDUSessionResourceSetupRequestTransfer, &pduSessionResourceSetupRequestTransfer); err != nil {
			g.NgapLog.Errorf("Error unmarshal pdu session resource setup request transfer: %v", err)
			return
		}
		g.NgapLog.Tracef("Get PDU Session Resource Setup Request Transfer: %+v", pduSessionResourceSetupRequestTransfer)
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

	if pduSessionResourceSetupRequestTransfer.ProtocolIEs != nil {
		for _, item := range pduSessionResourceSetupRequestTransfer.ProtocolIEs.List {
			if item.ULNGUUPTNLInformation == nil {
				continue
			}
			if gtpTunnel, ok := item.ULNGUUPTNLInformation.Choice.(*ie.GTPTunnel); ok && gtpTunnel.GTPTEID != nil {
				ranUe.SetUlTeid(gtpTunnel.GTPTEID.Value)
			}
		}
	}

	var (
		qosFlowPerTNLInformationItem ie.QosFlowPerTNLInformationItem
		err                          error
	)
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

func (d *ngapDispatcher) ueContextReleaseProcessor(g *Gnb, msg *message.UEContextReleaseCommand) {
	if msg.UENGAPIDs == nil {
		g.NgapLog.Errorf("Error ue context release: missing mandatory IE")
		return
	}

	uePair, ok := msg.UENGAPIDs.Choice.(*ie.UENGAPIDPair)
	if !ok || uePair.AMFUENGAPID == nil || uePair.RANUENGAPID == nil {
		g.NgapLog.Errorf("Error ue context release: unexpected UENGAPIDs choice")
		return
	}
	amfUeNgapId, ranUeNgapId := uePair.AMFUENGAPID.Value, uePair.RANUENGAPID.Value

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

	ngapUeContextReleaseCompleteMessage, err := getNgapUeContextReleaseCompleteMessage(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), []int64{constant.PDU_SESSION_ID}, g.plmnId, g.tai, g.gnbId)
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

func (d *ngapDispatcher) pduSessionResourceModifyIndicationProcessor(g *Gnb, msg *message.PDUSessionResourceModifyConfirm, ngapRaw []byte) {
	if msg.AMFUENGAPID == nil || msg.RANUENGAPID == nil {
		g.NgapLog.Errorf("Error pdu session resource modify indication: missing mandatory IE")
		return
	}

	amfUeNgapId := msg.AMFUENGAPID.Value
	ranUeNgapId := msg.RANUENGAPID.Value

	if msg.PDUSessionResourceFailedToModifyListModCfm != nil {
		g.NgapLog.Errorf("ran ue with ranUeNgapId %d pdu session resource modify indication failed", ranUeNgapId)
		return
	}
	if msg.PDUSessionResourceModifyListModCfm != nil {
		g.NgapLog.Infof("ran ue with ranUeNgapId %d pdu session resource modify indication successful", ranUeNgapId)
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
