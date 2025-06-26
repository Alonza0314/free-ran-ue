package gnb

import (
	"errors"
	"fmt"

	"github.com/free5gc/aper"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/sctp"
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

func ngapSetup(n2Conn *sctp.SCTPConn, gnbId []byte, gnbName string, plmnId ngapType.PLMNIdentity, tai ngapType.TAI, snssai ngapType.SNSSAI) error {
	request, err := getNgapSetupRequest(gnbId, gnbName, plmnId, tai, snssai)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting NGAP setup request: %v", err))
	}

	_, err = n2Conn.Write(request)
	if err != nil {
		return errors.New(fmt.Sprintf("Error sending NGAP setup request: %v", err))
	}

	response := make([]byte, 2048)
	responseLen, err := n2Conn.Read(response)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading NGAP setup response: %v", err))
	}

	responsePdu, err := ngap.Decoder(response[:responseLen])
	if err != nil {
		return errors.New(fmt.Sprintf("Error decoding NGAP setup response: %v", err))
	}

	if responsePdu.Present == ngapType.NGAPPDUPresentSuccessfulOutcome && responsePdu.SuccessfulOutcome.ProcedureCode.Value == ngapType.ProcedureCodeNGSetup {
		return nil
	}

	return errors.New(fmt.Sprintf("Error NGAP setup response: %+v", responsePdu))
}
