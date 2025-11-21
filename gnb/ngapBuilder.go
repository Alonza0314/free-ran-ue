package gnb

import (
	"fmt"

	"github.com/free5gc/aper"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
)

func buildNgapSetupRequest(gnbId []byte, gnbName string, plmnId ngapType.PLMNIdentity, tai ngapType.TAI, snssai ngapType.SNSSAI) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentInitiatingMessage
	pdu.InitiatingMessage = new(ngapType.InitiatingMessage)

	initiatingMessage := pdu.InitiatingMessage
	initiatingMessage.ProcedureCode.Value = ngapType.ProcedureCodeNGSetup
	initiatingMessage.Criticality.Value = ngapType.CriticalityPresentReject

	initiatingMessage.Value.Present = ngapType.InitiatingMessagePresentNGSetupRequest
	initiatingMessage.Value.NGSetupRequest = new(ngapType.NGSetupRequest)

	nGSetupRequest := initiatingMessage.Value.NGSetupRequest
	nGSetupRequestIEs := &nGSetupRequest.ProtocolIEs

	ie := ngapType.NGSetupRequestIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDGlobalRANNodeID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.NGSetupRequestIEsPresentGlobalRANNodeID
	ie.Value.GlobalRANNodeID = new(ngapType.GlobalRANNodeID)

	globalRANNodeID := ie.Value.GlobalRANNodeID
	globalRANNodeID.Present = ngapType.GlobalRANNodeIDPresentGlobalGNBID
	globalRANNodeID.GlobalGNBID = new(ngapType.GlobalGNBID)

	globalGNBID := globalRANNodeID.GlobalGNBID
	globalGNBID.PLMNIdentity.Value = plmnId.Value
	globalGNBID.GNBID.Present = ngapType.GNBIDPresentGNBID
	globalGNBID.GNBID.GNBID = new(aper.BitString)

	gNBID := globalGNBID.GNBID.GNBID
	*gNBID = aper.BitString{
		Bytes:     []byte(gnbId),
		BitLength: uint64(len(gnbId) * 8),
	}

	nGSetupRequestIEs.List = append(nGSetupRequestIEs.List, ie)

	ie = ngapType.NGSetupRequestIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANNodeName
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.NGSetupRequestIEsPresentRANNodeName
	ie.Value.RANNodeName = new(ngapType.RANNodeName)

	rANNodeName := ie.Value.RANNodeName
	rANNodeName.Value = gnbName
	nGSetupRequestIEs.List = append(nGSetupRequestIEs.List, ie)

	ie = ngapType.NGSetupRequestIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDSupportedTAList
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.NGSetupRequestIEsPresentSupportedTAList
	ie.Value.SupportedTAList = new(ngapType.SupportedTAList)

	supportedTAList := ie.Value.SupportedTAList

	supportedTAItem := ngapType.SupportedTAItem{}
	supportedTAItem.TAC.Value = aper.OctetString(tai.TAC.Value)

	broadcastPLMNList := &supportedTAItem.BroadcastPLMNList
	broadcastPLMNItem := ngapType.BroadcastPLMNItem{}
	broadcastPLMNItem.PLMNIdentity.Value = tai.PLMNIdentity.Value

	sliceSupportList := &broadcastPLMNItem.TAISliceSupportList
	sliceSupportItem := ngapType.SliceSupportItem{}
	sliceSupportItem.SNSSAI.SST.Value = aper.OctetString(snssai.SST.Value)
	sliceSupportItem.SNSSAI.SD = new(ngapType.SD)
	sliceSupportItem.SNSSAI.SD.Value = aper.OctetString(snssai.SD.Value)

	sliceSupportList.List = append(sliceSupportList.List, sliceSupportItem)

	broadcastPLMNList.List = append(broadcastPLMNList.List, broadcastPLMNItem)

	supportedTAList.List = append(supportedTAList.List, supportedTAItem)

	nGSetupRequestIEs.List = append(nGSetupRequestIEs.List, ie)

	ie = ngapType.NGSetupRequestIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDDefaultPagingDRX
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.NGSetupRequestIEsPresentDefaultPagingDRX
	ie.Value.DefaultPagingDRX = new(ngapType.PagingDRX)

	pagingDRX := ie.Value.DefaultPagingDRX
	pagingDRX.Value = ngapType.PagingDRXPresentV128
	nGSetupRequestIEs.List = append(nGSetupRequestIEs.List, ie)

	return pdu
}

func getNgapSetupRequest(gnbId []byte, gnbName string, plmnId ngapType.PLMNIdentity, tai ngapType.TAI, snssai ngapType.SNSSAI) ([]byte, error) {
	return ngap.Encoder(buildNgapSetupRequest(gnbId, gnbName, plmnId, tai, snssai))
}

func buildInitialUeMessage(ranUeNgapId int64, ueRegistrationRequest []byte, plmnId ngapType.PLMNIdentity, tai ngapType.TAI) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentInitiatingMessage
	pdu.InitiatingMessage = new(ngapType.InitiatingMessage)

	initiatingMessage := pdu.InitiatingMessage
	initiatingMessage.ProcedureCode.Value = ngapType.ProcedureCodeInitialUEMessage
	initiatingMessage.Criticality.Value = ngapType.CriticalityPresentIgnore

	initiatingMessage.Value.Present = ngapType.InitiatingMessagePresentInitialUEMessage
	initiatingMessage.Value.InitialUEMessage = new(ngapType.InitialUEMessage)

	initialUEMessage := initiatingMessage.Value.InitialUEMessage
	initialUEMessageIEs := &initialUEMessage.ProtocolIEs

	// RAN UE NGAP ID
	ie := ngapType.InitialUEMessageIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.InitialUEMessageIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)

	rANUENGAPID := ie.Value.RANUENGAPID
	rANUENGAPID.Value = ranUeNgapId

	initialUEMessageIEs.List = append(initialUEMessageIEs.List, ie)

	// NAS PDU
	ie = ngapType.InitialUEMessageIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDNASPDU
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.InitialUEMessageIEsPresentNASPDU
	ie.Value.NASPDU = new(ngapType.NASPDU)

	nasPDU := ie.Value.NASPDU
	nasPDU.Value = ueRegistrationRequest

	initialUEMessageIEs.List = append(initialUEMessageIEs.List, ie)

	// User Location Information
	ie = ngapType.InitialUEMessageIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDUserLocationInformation
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.InitialUEMessageIEsPresentUserLocationInformation
	ie.Value.UserLocationInformation = new(ngapType.UserLocationInformation)

	userLocationInformation := ie.Value.UserLocationInformation
	userLocationInformation.Present = ngapType.UserLocationInformationPresentUserLocationInformationNR
	userLocationInformation.UserLocationInformationNR = new(ngapType.UserLocationInformationNR)

	userLocationInformationNR := userLocationInformation.UserLocationInformationNR
	userLocationInformationNR.NRCGI.PLMNIdentity.Value = plmnId.Value
	userLocationInformationNR.NRCGI.NRCellIdentity.Value = aper.BitString{
		Bytes:     []byte{0x00, 0x00, 0x00, 0x00, 0x10},
		BitLength: 36,
	}
	userLocationInformationNR.TAI.PLMNIdentity.Value = tai.PLMNIdentity.Value
	userLocationInformationNR.TAI.TAC.Value = tai.TAC.Value

	initialUEMessageIEs.List = append(initialUEMessageIEs.List, ie)

	// RRC Establishment Cause
	ie = ngapType.InitialUEMessageIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRRCEstablishmentCause
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.InitialUEMessageIEsPresentRRCEstablishmentCause
	ie.Value.RRCEstablishmentCause = new(ngapType.RRCEstablishmentCause)

	rRCEstablishmentCause := ie.Value.RRCEstablishmentCause
	rRCEstablishmentCause.Value = ngapType.RRCEstablishmentCausePresentMtAccess

	initialUEMessageIEs.List = append(initialUEMessageIEs.List, ie)

	// UE Context Request
	ie = ngapType.InitialUEMessageIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDUEContextRequest
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.InitialUEMessageIEsPresentUEContextRequest
	ie.Value.UEContextRequest = new(ngapType.UEContextRequest)
	ie.Value.UEContextRequest.Value = ngapType.UEContextRequestPresentRequested
	initialUEMessageIEs.List = append(initialUEMessageIEs.List, ie)

	return pdu
}

func getInitialUeMessage(ranUeNgapId int64, ueRegistrationRequest []byte, plmnId ngapType.PLMNIdentity, tai ngapType.TAI) ([]byte, error) {
	initialUeMessage := buildInitialUeMessage(ranUeNgapId, ueRegistrationRequest, plmnId, tai)
	return ngap.Encoder(initialUeMessage)
}

func buildUplinkNasTransport(amfUeNgapId int64, ranUeNgapId int64, plmnId ngapType.PLMNIdentity, tai ngapType.TAI, nasPdu []byte) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentInitiatingMessage
	pdu.InitiatingMessage = new(ngapType.InitiatingMessage)

	initiatingMessage := pdu.InitiatingMessage
	initiatingMessage.ProcedureCode.Value = ngapType.ProcedureCodeUplinkNASTransport
	initiatingMessage.Criticality.Value = ngapType.CriticalityPresentIgnore

	initiatingMessage.Value.Present = ngapType.InitiatingMessagePresentUplinkNASTransport
	initiatingMessage.Value.UplinkNASTransport = new(ngapType.UplinkNASTransport)

	uplinkNasTransport := initiatingMessage.Value.UplinkNASTransport
	uplinkNasTransportIEs := &uplinkNasTransport.ProtocolIEs

	// AMF UE NGAP ID
	ie := ngapType.UplinkNASTransportIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.UplinkNASTransportIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)

	aMFUENGAPID := ie.Value.AMFUENGAPID
	aMFUENGAPID.Value = amfUeNgapId

	uplinkNasTransportIEs.List = append(uplinkNasTransportIEs.List, ie)

	// RAN UE NGAP ID
	ie = ngapType.UplinkNASTransportIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.UplinkNASTransportIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)

	rANUENGAPID := ie.Value.RANUENGAPID
	rANUENGAPID.Value = ranUeNgapId

	uplinkNasTransportIEs.List = append(uplinkNasTransportIEs.List, ie)

	// NAS-PDU
	ie = ngapType.UplinkNASTransportIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDNASPDU
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.UplinkNASTransportIEsPresentNASPDU
	ie.Value.NASPDU = new(ngapType.NASPDU)

	nASPDU := ie.Value.NASPDU
	nASPDU.Value = nasPdu

	uplinkNasTransportIEs.List = append(uplinkNasTransportIEs.List, ie)

	// User Location Information
	ie = ngapType.UplinkNASTransportIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDUserLocationInformation
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.UplinkNASTransportIEsPresentUserLocationInformation
	ie.Value.UserLocationInformation = new(ngapType.UserLocationInformation)

	userLocationInformation := ie.Value.UserLocationInformation
	userLocationInformation.Present = ngapType.UserLocationInformationPresentUserLocationInformationNR
	userLocationInformation.UserLocationInformationNR = new(ngapType.UserLocationInformationNR)

	userLocationInformationNR := userLocationInformation.UserLocationInformationNR
	userLocationInformationNR.NRCGI.PLMNIdentity.Value = plmnId.Value
	userLocationInformationNR.NRCGI.NRCellIdentity.Value = aper.BitString{
		Bytes:     []byte{0x00, 0x00, 0x00, 0x00, 0x10},
		BitLength: 36,
	}

	userLocationInformationNR.TAI.PLMNIdentity.Value = tai.PLMNIdentity.Value
	userLocationInformationNR.TAI.TAC.Value = tai.TAC.Value

	uplinkNasTransportIEs.List = append(uplinkNasTransportIEs.List, ie)

	return pdu
}

func getUplinkNasTransport(amfUeNgapId int64, ranUeNgapId int64, plmnId ngapType.PLMNIdentity, tai ngapType.TAI, nasPdu []byte) ([]byte, error) {
	uplinkNasTransport := buildUplinkNasTransport(amfUeNgapId, ranUeNgapId, plmnId, tai, nasPdu)
	return ngap.Encoder(uplinkNasTransport)
}

func buildNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId int64) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentSuccessfulOutcome
	pdu.SuccessfulOutcome = new(ngapType.SuccessfulOutcome)

	successfulOutcome := pdu.SuccessfulOutcome
	successfulOutcome.ProcedureCode.Value = ngapType.ProcedureCodeInitialContextSetup
	successfulOutcome.Criticality.Value = ngapType.CriticalityPresentReject

	successfulOutcome.Value.Present = ngapType.SuccessfulOutcomePresentInitialContextSetupResponse
	successfulOutcome.Value.InitialContextSetupResponse = new(ngapType.InitialContextSetupResponse)

	initialContextSetupResponse := successfulOutcome.Value.InitialContextSetupResponse
	initialContextSetupResponseIEs := &initialContextSetupResponse.ProtocolIEs

	// AMF UE NGAP ID
	ie := ngapType.InitialContextSetupResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.InitialContextSetupResponseIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)

	aMFUENGAPID := ie.Value.AMFUENGAPID
	aMFUENGAPID.Value = amfUeNgapId

	initialContextSetupResponseIEs.List = append(initialContextSetupResponseIEs.List, ie)

	// RAN UE NGAP ID
	ie = ngapType.InitialContextSetupResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.InitialContextSetupResponseIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)

	rANUENGAPID := ie.Value.RANUENGAPID
	rANUENGAPID.Value = ranUeNgapId

	initialContextSetupResponseIEs.List = append(initialContextSetupResponseIEs.List, ie)

	return pdu
}

func getNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId int64) ([]byte, error) {
	initialContextSetupResponse := buildNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId)
	return ngap.Encoder(initialContextSetupResponse)
}

func buildPduSessionResourceSetupResponseTransfer(dlTeid []byte, ranN3Ip string, qosId int64, nrdcIndicator bool, qosFlowPerTNLInformationItem ngapType.QosFlowPerTNLInformationItem) ngapType.PDUSessionResourceSetupResponseTransfer {
	transferMessage := ngapType.PDUSessionResourceSetupResponseTransfer{}

	// QoS Flow per TNL Information
	qosFlowPerTNLInformation := &transferMessage.DLQosFlowPerTNLInformation
	qosFlowPerTNLInformation.UPTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel

	// UP Transport Layer Information in QoS Flow per TNL Information
	upTransportLayerInformation := &qosFlowPerTNLInformation.UPTransportLayerInformation
	upTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	upTransportLayerInformation.GTPTunnel = new(ngapType.GTPTunnel)
	upTransportLayerInformation.GTPTunnel.GTPTEID.Value = aper.OctetString(dlTeid)
	upTransportLayerInformation.GTPTunnel.TransportLayerAddress = ngapConvert.IPAddressToNgap(ranN3Ip, "")

	// Associated QoS Flow List in QoS Flow per TNL Information
	associatedQosFlowList := &qosFlowPerTNLInformation.AssociatedQosFlowList

	associatedQosFlowItem := ngapType.AssociatedQosFlowItem{}
	associatedQosFlowItem.QosFlowIdentifier.Value = qosId
	associatedQosFlowList.List = append(associatedQosFlowList.List, associatedQosFlowItem)

	if nrdcIndicator && qosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation.Present == ngapType.UPTransportLayerInformationPresentGTPTunnel && qosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation.GTPTunnel.GTPTEID.Value != nil {
		transferMessage.AdditionalDLQosFlowPerTNLInformation = new(ngapType.QosFlowPerTNLInformationList)
		transferMessage.AdditionalDLQosFlowPerTNLInformation.List = append(transferMessage.AdditionalDLQosFlowPerTNLInformation.List, qosFlowPerTNLInformationItem)
	}

	return transferMessage
}

func getPduSessionResourceSetupResponseTransfer(dlTeid []byte, ranN3Ip string, qosId int64, nrdcIndicator bool, qosFlowPerTNLInformationItem ngapType.QosFlowPerTNLInformationItem) ([]byte, error) {
	transferMessage := buildPduSessionResourceSetupResponseTransfer(dlTeid, ranN3Ip, qosId, nrdcIndicator, qosFlowPerTNLInformationItem)
	encodedTransferMessage, err := aper.MarshalWithParams(transferMessage, "valueExt")
	if err != nil {
		return nil, fmt.Errorf("error marshal pdu session resource setup response transfer message: %v", err)
	}
	return encodedTransferMessage, nil
}

func buildPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId int64, pduSessionResourceSetupResponseTransferMessage []byte) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentSuccessfulOutcome
	pdu.SuccessfulOutcome = new(ngapType.SuccessfulOutcome)

	successfulOutcome := pdu.SuccessfulOutcome
	successfulOutcome.ProcedureCode.Value = ngapType.ProcedureCodePDUSessionResourceSetup
	successfulOutcome.Criticality.Value = ngapType.CriticalityPresentReject

	successfulOutcome.Value.Present = ngapType.SuccessfulOutcomePresentPDUSessionResourceSetupResponse
	successfulOutcome.Value.PDUSessionResourceSetupResponse = new(ngapType.PDUSessionResourceSetupResponse)

	pDUSessionResourceSetupResponse := successfulOutcome.Value.PDUSessionResourceSetupResponse
	pDUSessionResourceSetupResponseIEs := &pDUSessionResourceSetupResponse.ProtocolIEs

	// AMF UE NGAP ID
	ie := ngapType.PDUSessionResourceSetupResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceSetupResponseIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)

	aMFUENGAPID := ie.Value.AMFUENGAPID
	aMFUENGAPID.Value = amfUeNgapId

	pDUSessionResourceSetupResponseIEs.List = append(pDUSessionResourceSetupResponseIEs.List, ie)

	// RAN UE NGAP ID
	ie = ngapType.PDUSessionResourceSetupResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceSetupResponseIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)

	rANUENGAPID := ie.Value.RANUENGAPID
	rANUENGAPID.Value = ranUeNgapId

	pDUSessionResourceSetupResponseIEs.List = append(pDUSessionResourceSetupResponseIEs.List, ie)

	// PDU Session Resource Setup Response List
	ie = ngapType.PDUSessionResourceSetupResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDPDUSessionResourceSetupListSURes
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceSetupResponseIEsPresentPDUSessionResourceSetupListSURes
	ie.Value.PDUSessionResourceSetupListSURes = new(ngapType.PDUSessionResourceSetupListSURes)

	pDUSessionResourceSetupListSURes := ie.Value.PDUSessionResourceSetupListSURes

	// PDU Session Resource Setup Response Item in PDU Session Resource Setup Response List
	pDUSessionResourceSetupItemSURes := ngapType.PDUSessionResourceSetupItemSURes{}
	pDUSessionResourceSetupItemSURes.PDUSessionID.Value = pduSessionId

	pDUSessionResourceSetupItemSURes.PDUSessionResourceSetupResponseTransfer = pduSessionResourceSetupResponseTransferMessage

	pDUSessionResourceSetupListSURes.List = append(pDUSessionResourceSetupListSURes.List, pDUSessionResourceSetupItemSURes)

	pDUSessionResourceSetupResponseIEs.List = append(pDUSessionResourceSetupResponseIEs.List, ie)

	return pdu
}

func getPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId int64, pduSessionResourceSetupResponseTransferMessage []byte) ([]byte, error) {
	pduSessionResourceSetupResponse := buildPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId, pduSessionResourceSetupResponseTransferMessage)
	return ngap.Encoder(pduSessionResourceSetupResponse)
}

func buildNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId int64, pduSessionIdList []int64, plmnId ngapType.PLMNIdentity, tai ngapType.TAI) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentSuccessfulOutcome
	pdu.SuccessfulOutcome = new(ngapType.SuccessfulOutcome)

	successfulOutcome := pdu.SuccessfulOutcome
	successfulOutcome.ProcedureCode.Value = ngapType.ProcedureCodeUEContextRelease
	successfulOutcome.Criticality.Value = ngapType.CriticalityPresentReject

	successfulOutcome.Value.Present = ngapType.SuccessfulOutcomePresentUEContextReleaseComplete
	successfulOutcome.Value.UEContextReleaseComplete = new(ngapType.UEContextReleaseComplete)

	uEContextReleaseComplete := successfulOutcome.Value.UEContextReleaseComplete
	uEContextReleaseCompleteIEs := &uEContextReleaseComplete.ProtocolIEs

	// AMF UE NGAP ID
	ie := ngapType.UEContextReleaseCompleteIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.UEContextReleaseCompleteIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)

	aMFUENGAPID := ie.Value.AMFUENGAPID
	aMFUENGAPID.Value = amfUeNgapId

	uEContextReleaseCompleteIEs.List = append(uEContextReleaseCompleteIEs.List, ie)

	// RAN UE NGAP ID
	ie = ngapType.UEContextReleaseCompleteIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.UEContextReleaseCompleteIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)

	rANUENGAPID := ie.Value.RANUENGAPID
	rANUENGAPID.Value = ranUeNgapId

	uEContextReleaseCompleteIEs.List = append(uEContextReleaseCompleteIEs.List, ie)

	// User Location Information
	ie = ngapType.UEContextReleaseCompleteIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDUserLocationInformation
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.UEContextReleaseCompleteIEsPresentUserLocationInformation
	ie.Value.UserLocationInformation = new(ngapType.UserLocationInformation)

	userLocationInformation := ie.Value.UserLocationInformation
	userLocationInformation.Present = ngapType.UserLocationInformationPresentUserLocationInformationNR
	userLocationInformation.UserLocationInformationNR = new(ngapType.UserLocationInformationNR)

	userLocationInformationNR := userLocationInformation.UserLocationInformationNR
	userLocationInformationNR.NRCGI.PLMNIdentity.Value = plmnId.Value
	userLocationInformationNR.NRCGI.NRCellIdentity.Value = aper.BitString{
		Bytes:     []byte{0x00, 0x00, 0x00, 0x00, 0x10},
		BitLength: 36,
	}

	userLocationInformationNR.TAI.PLMNIdentity.Value = tai.PLMNIdentity.Value
	userLocationInformationNR.TAI.TAC.Value = tai.TAC.Value

	uEContextReleaseCompleteIEs.List = append(uEContextReleaseCompleteIEs.List, ie)

	if len(pduSessionIdList) > 0 {
		ie = ngapType.UEContextReleaseCompleteIEs{}
		ie.Id.Value = ngapType.ProtocolIEIDPDUSessionResourceListCxtRelCpl
		ie.Criticality.Value = ngapType.CriticalityPresentReject
		ie.Value.Present = ngapType.UEContextReleaseCompleteIEsPresentPDUSessionResourceListCxtRelCpl
		ie.Value.PDUSessionResourceListCxtRelCpl = new(ngapType.PDUSessionResourceListCxtRelCpl)

		pDUSessionResourceListCxtRelCpl := ie.Value.PDUSessionResourceListCxtRelCpl

		// PDU Session Resource Item (in PDU Session Resource List)
		for _, pduSessionID := range pduSessionIdList {
			pDUSessionResourceItemCxtRelCpl := ngapType.PDUSessionResourceItemCxtRelCpl{}
			pDUSessionResourceItemCxtRelCpl.PDUSessionID.Value = pduSessionID
			pDUSessionResourceListCxtRelCpl.List = append(pDUSessionResourceListCxtRelCpl.List, pDUSessionResourceItemCxtRelCpl)
		}

		uEContextReleaseCompleteIEs.List = append(uEContextReleaseCompleteIEs.List, ie)
	}

	return pdu
}

func getNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId int64, pduSessionIdList []int64, plmnId ngapType.PLMNIdentity, tai ngapType.TAI) ([]byte, error) {
	ngapUeContextReleaseComplete := buildNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId, pduSessionIdList, plmnId, tai)
	return ngap.Encoder(ngapUeContextReleaseComplete)
}

func buildPDUSessionResourceModifyIndicationTransfer(dlTeid []byte, ranN3Ip string, qosId int64) ngapType.PDUSessionResourceModifyIndicationTransfer {
	transferMessage := ngapType.PDUSessionResourceModifyIndicationTransfer{}

	// QoS Flow per TNL Information
	qosFlowPerTNLInformation := &transferMessage.DLQosFlowPerTNLInformation
	qosFlowPerTNLInformation.UPTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel

	// UP Transport Layer Information in QoS Flow per TNL Information
	upTransportLayerInformation := &qosFlowPerTNLInformation.UPTransportLayerInformation
	upTransportLayerInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	upTransportLayerInformation.GTPTunnel = new(ngapType.GTPTunnel)
	upTransportLayerInformation.GTPTunnel.GTPTEID.Value = aper.OctetString(dlTeid)
	upTransportLayerInformation.GTPTunnel.TransportLayerAddress = ngapConvert.IPAddressToNgap(ranN3Ip, "")

	// Associated QoS Flow List in QoS Flow per TNL Information
	associatedQosFlowList := &qosFlowPerTNLInformation.AssociatedQosFlowList

	associatedQosFlowItem := ngapType.AssociatedQosFlowItem{}
	associatedQosFlowItem.QosFlowIdentifier.Value = qosId
	associatedQosFlowList.List = append(associatedQosFlowList.List, associatedQosFlowItem)

	return transferMessage
}

func getPDUSessionResourceModifyIndicationTransfer(dlTeid []byte, ranN3Ip string, qosId int64) ([]byte, error) {
	transferMessage := buildPDUSessionResourceModifyIndicationTransfer(dlTeid, ranN3Ip, qosId)
	encodedTransferMessage, err := aper.MarshalWithParams(transferMessage, "valueExt")
	if err != nil {
		return nil, fmt.Errorf("error marshal pdu session resource modify indication transfer message: %v", err)
	}
	return encodedTransferMessage, nil
}

func buildPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId int64, pduSessionId int64, pduSessionResourceModifyIndicationTransferMessage []byte) ngapType.NGAPPDU {
	pdu := ngapType.NGAPPDU{}

	pdu.Present = ngapType.NGAPPDUPresentInitiatingMessage
	pdu.InitiatingMessage = new(ngapType.InitiatingMessage)

	initiatingMessage := pdu.InitiatingMessage
	initiatingMessage.ProcedureCode.Value = ngapType.ProcedureCodePDUSessionResourceModifyIndication
	initiatingMessage.Criticality.Value = ngapType.CriticalityPresentReject
	
	initiatingMessage.Value.Present = ngapType.InitiatingMessagePresentPDUSessionResourceModifyIndication
	initiatingMessage.Value.PDUSessionResourceModifyIndication = new(ngapType.PDUSessionResourceModifyIndication)

	indication := initiatingMessage.Value.PDUSessionResourceModifyIndication
	indicationIEs := &indication.ProtocolIEs

	// AMF UE NGAP ID
	ie := ngapType.PDUSessionResourceModifyIndicationIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceModifyIndicationIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)
	ie.Value.AMFUENGAPID.Value = amfUeNgapId
	indicationIEs.List = append(indicationIEs.List, ie)

	// RAN UE NGAP ID
	ie = ngapType.PDUSessionResourceModifyIndicationIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceModifyIndicationIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)
	ie.Value.RANUENGAPID.Value = ranUeNgapId
	indicationIEs.List = append(indicationIEs.List, ie)

	// PDU Session Resource Modify List
	ie = ngapType.PDUSessionResourceModifyIndicationIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDPDUSessionResourceModifyListModInd
	ie.Criticality.Value = ngapType.CriticalityPresentReject
	ie.Value.Present = ngapType.PDUSessionResourceModifyIndicationIEsPresentPDUSessionResourceModifyListModInd
	ie.Value.PDUSessionResourceModifyListModInd = new(ngapType.PDUSessionResourceModifyListModInd)

	modifyItem := ngapType.PDUSessionResourceModifyItemModInd{}
	modifyItem.PDUSessionID.Value = pduSessionId
	modifyItem.PDUSessionResourceModifyIndicationTransfer = pduSessionResourceModifyIndicationTransferMessage

	ie.Value.PDUSessionResourceModifyListModInd.List = append(ie.Value.PDUSessionResourceModifyListModInd.List, modifyItem)

	indicationIEs.List = append(indicationIEs.List, ie)

	return pdu
}

func getPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId int64, pduSessionId int64, pduSessionResourceModifyIndicationTransferMessage []byte) ([]byte, error) {
	pduSessionResourceModifyIndication := buildPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId, pduSessionId, pduSessionResourceModifyIndicationTransferMessage)
	return ngap.Encoder(pduSessionResourceModifyIndication)
}