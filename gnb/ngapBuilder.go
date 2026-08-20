package gnb

import (
	"fmt"
	"net"

	"github.com/free5gc/ngap/aper"
	"github.com/free5gc/ngap/ie"
	"github.com/free5gc/ngap/message"
)

// ngapConvertIPAddressToNgap builds an aper.BitString TransportLayerAddress
// value from an IPv4 address, per 3GPP TS 38.414.
func ngapConvertIPAddressToNgap(ipv4Addr string) aper.BitString {
	ipv4NetIP := net.ParseIP(ipv4Addr).To4()
	return aper.BitString{
		Bytes:     []byte{ipv4NetIP[0], ipv4NetIP[1], ipv4NetIP[2], ipv4NetIP[3]},
		BitLength: 32,
	}
}

func buildNRCellIdentityFromGnbID(gnbId []byte) aper.BitString {
	const nrCellIdentityBits = 36

	var gnbIDValue uint64
	for _, b := range gnbId {
		gnbIDValue = (gnbIDValue << 8) | uint64(b)
	}

	gnbIDBits := len(gnbId) * 8
	if gnbIDBits > nrCellIdentityBits {
		gnbIDValue >>= uint(gnbIDBits - nrCellIdentityBits)
		gnbIDBits = nrCellIdentityBits
	}

	cellIDBits := nrCellIdentityBits - gnbIDBits
	var nrCellIdentityValue uint64
	if cellIDBits > 0 {
		nrCellIdentityValue = (gnbIDValue << uint(cellIDBits)) | 1
	} else {
		nrCellIdentityValue = gnbIDValue
	}

	packed := nrCellIdentityValue << 4
	return aper.BitString{
		Bytes: []byte{
			byte((packed >> 32) & 0xff),
			byte((packed >> 24) & 0xff),
			byte((packed >> 16) & 0xff),
			byte((packed >> 8) & 0xff),
			byte(packed & 0xff),
		},
		BitLength: nrCellIdentityBits,
	}
}

func buildUserLocationInformationNR(plmnId ie.PLMNIdentity, tai ie.TAI, gnbId []byte) *ie.UserLocationInformation {
	return &ie.UserLocationInformation{
		Choice: &ie.UserLocationInformationNR{
			NRCGI: &ie.NRCGI{
				PLMNIdentity: &plmnId,
				NRCellIdentity: &ie.NRCellIdentity{
					Value: buildNRCellIdentityFromGnbID(gnbId),
				},
			},
			TAI: &ie.TAI{
				PLMNIdentity: tai.PLMNIdentity,
				TAC:          tai.TAC,
			},
		},
	}
}

func buildNgapSetupRequest(gnbId []byte, gnbName string, plmnId ie.PLMNIdentity, tai ie.TAI, snssai ie.SNSSAI) *message.NGSetupRequest {
	return &message.NGSetupRequest{
		GlobalRANNodeID: &ie.GlobalRANNodeID{
			Choice: &ie.GlobalGNBID{
				PLMNIdentity: &plmnId,
				GNBID: &ie.GNBID{
					Choice: &ie.GNBIDForGNBID{
						Value: aper.BitString{
							Bytes:     gnbId,
							BitLength: uint64(len(gnbId) * 8),
						},
					},
				},
			},
		},
		RANNodeName: &ie.RANNodeName{
			Value: aper.PrintableString(gnbName),
		},
		SupportedTAList: &ie.SupportedTAList{
			List: []ie.SupportedTAItem{
				{
					TAC: tai.TAC,
					BroadcastPLMNList: &ie.BroadcastPLMNList{
						List: []ie.BroadcastPLMNItem{
							{
								PLMNIdentity: tai.PLMNIdentity,
								TAISliceSupportList: &ie.SliceSupportList{
									List: []ie.SliceSupportItem{
										{SNSSAI: &snssai},
									},
								},
							},
						},
					},
				},
			},
		},
		DefaultPagingDRX: &ie.PagingDRX{
			Value: ie.PagingDRXPresentV128,
		},
	}
}

func getNgapSetupRequest(gnbId []byte, gnbName string, plmnId ie.PLMNIdentity, tai ie.TAI, snssai ie.SNSSAI) ([]byte, error) {
	return buildNgapSetupRequest(gnbId, gnbName, plmnId, tai, snssai).MarshalBinary()
}

func buildInitialUeMessage(ranUeNgapId int64, ueRegistrationRequest []byte, plmnId ie.PLMNIdentity, tai ie.TAI, gnbId []byte) *message.InitialUEMessage {
	return &message.InitialUEMessage{
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
		NASPDU: &ie.NASPDU{
			Value: aper.OctetString(ueRegistrationRequest),
		},
		UserLocationInformation: buildUserLocationInformationNR(plmnId, tai, gnbId),
		RRCEstablishmentCause: &ie.RRCEstablishmentCause{
			Value: ie.RRCEstablishmentCausePresentMtAccess,
		},
		UEContextRequest: &ie.UEContextRequest{
			Value: ie.UEContextRequestPresentRequested,
		},
	}
}

func getInitialUeMessage(ranUeNgapId int64, ueRegistrationRequest []byte, plmnId ie.PLMNIdentity, tai ie.TAI, gnbId []byte) ([]byte, error) {
	return buildInitialUeMessage(ranUeNgapId, ueRegistrationRequest, plmnId, tai, gnbId).MarshalBinary()
}

func buildUplinkNasTransport(amfUeNgapId int64, ranUeNgapId int64, plmnId ie.PLMNIdentity, tai ie.TAI, nasPdu []byte, gnbId []byte) *message.UplinkNASTransport {
	return &message.UplinkNASTransport{
		AMFUENGAPID: &ie.AMFUENGAPID{
			Value: amfUeNgapId,
		},
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
		NASPDU: &ie.NASPDU{
			Value: aper.OctetString(nasPdu),
		},
		UserLocationInformation: buildUserLocationInformationNR(plmnId, tai, gnbId),
	}
}

func getUplinkNasTransport(amfUeNgapId int64, ranUeNgapId int64, plmnId ie.PLMNIdentity, tai ie.TAI, nasPdu []byte, gnbId []byte) ([]byte, error) {
	return buildUplinkNasTransport(amfUeNgapId, ranUeNgapId, plmnId, tai, nasPdu, gnbId).MarshalBinary()
}

func buildNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId int64) *message.InitialContextSetupResponse {
	return &message.InitialContextSetupResponse{
		AMFUENGAPID: &ie.AMFUENGAPID{
			Value: amfUeNgapId,
		},
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
	}
}

func getNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId int64) ([]byte, error) {
	return buildNgapInitialContextSetupResponse(amfUeNgapId, ranUeNgapId).MarshalBinary()
}

func buildPduSessionResourceSetupResponseTransfer(dlTeid []byte, ranN3Ip string, qosId int64, nrdcIndicator bool, qosFlowPerTNLInformationItem ie.QosFlowPerTNLInformationItem) ie.PDUSessionResourceSetupResponseTransfer {
	transferMessage := ie.PDUSessionResourceSetupResponseTransfer{
		DLQosFlowPerTNLInformation: &ie.QosFlowPerTNLInformation{
			UPTransportLayerInformation: &ie.UPTransportLayerInformation{
				Choice: &ie.GTPTunnel{
					GTPTEID: &ie.GTPTEID{
						Value: aper.OctetString(dlTeid),
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
							Value: qosId,
						},
					},
				},
			},
		},
	}

	if nrdcIndicator {
		if gtpTunnel, ok := qosFlowPerTNLInformationItem.QosFlowPerTNLInformation.UPTransportLayerInformation.Choice.(*ie.GTPTunnel); ok && gtpTunnel.GTPTEID != nil && gtpTunnel.GTPTEID.Value != nil {
			transferMessage.AdditionalDLQosFlowPerTNLInformation = &ie.QosFlowPerTNLInformationList{
				List: []ie.QosFlowPerTNLInformationItem{qosFlowPerTNLInformationItem},
			}
		}
	}

	return transferMessage
}

func getPduSessionResourceSetupResponseTransfer(dlTeid []byte, ranN3Ip string, qosId int64, nrdcIndicator bool, qosFlowPerTNLInformationItem ie.QosFlowPerTNLInformationItem) ([]byte, error) {
	transferMessage := buildPduSessionResourceSetupResponseTransfer(dlTeid, ranN3Ip, qosId, nrdcIndicator, qosFlowPerTNLInformationItem)
	encodedTransferMessage, err := ie.MarshalBinary(&transferMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshal pdu session resource setup response transfer message: %v", err)
	}
	return encodedTransferMessage, nil
}

func buildPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId int64, pduSessionResourceSetupResponseTransferMessage []byte) *message.PDUSessionResourceSetupResponse {
	transfer := aper.OctetString(pduSessionResourceSetupResponseTransferMessage)

	return &message.PDUSessionResourceSetupResponse{
		AMFUENGAPID: &ie.AMFUENGAPID{
			Value: amfUeNgapId,
		},
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
		PDUSessionResourceSetupListSURes: &ie.PDUSessionResourceSetupListSURes{
			List: []ie.PDUSessionResourceSetupItemSURes{
				{
					PDUSessionID: &ie.PDUSessionID{
						Value: pduSessionId,
					},
					PDUSessionResourceSetupResponseTransfer: &transfer,
				},
			},
		},
	}
}

func getPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId int64, pduSessionResourceSetupResponseTransferMessage []byte) ([]byte, error) {
	return buildPduSessionResourceSetupResponse(amfUeNgapId, ranUeNgapId, pduSessionId, pduSessionResourceSetupResponseTransferMessage).MarshalBinary()
}

func buildNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId int64, pduSessionIdList []int64, plmnId ie.PLMNIdentity, tai ie.TAI, gnbId []byte) *message.UEContextReleaseComplete {
	ueContextReleaseComplete := &message.UEContextReleaseComplete{
		AMFUENGAPID: &ie.AMFUENGAPID{
			Value: amfUeNgapId,
		},
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
		UserLocationInformation: buildUserLocationInformationNR(plmnId, tai, gnbId),
	}

	if len(pduSessionIdList) > 0 {
		pduSessionResourceListCxtRelCpl := &ie.PDUSessionResourceListCxtRelCpl{}
		for _, pduSessionID := range pduSessionIdList {
			pduSessionResourceListCxtRelCpl.List = append(pduSessionResourceListCxtRelCpl.List, ie.PDUSessionResourceItemCxtRelCpl{
				PDUSessionID: &ie.PDUSessionID{
					Value: pduSessionID,
				},
			})
		}
		ueContextReleaseComplete.PDUSessionResourceListCxtRelCpl = pduSessionResourceListCxtRelCpl
	}

	return ueContextReleaseComplete
}

func getNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId int64, pduSessionIdList []int64, plmnId ie.PLMNIdentity, tai ie.TAI, gnbId []byte) ([]byte, error) {
	return buildNgapUeContextReleaseCompleteMessage(amfUeNgapId, ranUeNgapId, pduSessionIdList, plmnId, tai, gnbId).MarshalBinary()
}

func buildPDUSessionResourceModifyIndicationTransfer(dlTeid []byte, ranN3Ip string, qosId int64) ie.PDUSessionResourceModifyIndicationTransfer {
	return ie.PDUSessionResourceModifyIndicationTransfer{
		DLQosFlowPerTNLInformation: &ie.QosFlowPerTNLInformation{
			UPTransportLayerInformation: &ie.UPTransportLayerInformation{
				Choice: &ie.GTPTunnel{
					GTPTEID: &ie.GTPTEID{
						Value: aper.OctetString(dlTeid),
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
							Value: qosId,
						},
					},
				},
			},
		},
	}
}

func getPDUSessionResourceModifyIndicationTransfer(dlTeid []byte, ranN3Ip string, qosId int64) ([]byte, error) {
	transferMessage := buildPDUSessionResourceModifyIndicationTransfer(dlTeid, ranN3Ip, qosId)
	encodedTransferMessage, err := ie.MarshalBinary(&transferMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshal pdu session resource modify indication transfer message: %v", err)
	}
	return encodedTransferMessage, nil
}

func buildPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId int64, pduSessionId int64, pduSessionResourceModifyIndicationTransferMessage []byte) *message.PDUSessionResourceModifyIndication {
	transfer := aper.OctetString(pduSessionResourceModifyIndicationTransferMessage)

	return &message.PDUSessionResourceModifyIndication{
		AMFUENGAPID: &ie.AMFUENGAPID{
			Value: amfUeNgapId,
		},
		RANUENGAPID: &ie.RANUENGAPID{
			Value: ranUeNgapId,
		},
		PDUSessionResourceModifyListModInd: &ie.PDUSessionResourceModifyListModInd{
			List: []ie.PDUSessionResourceModifyItemModInd{
				{
					PDUSessionID: &ie.PDUSessionID{
						Value: pduSessionId,
					},
					PDUSessionResourceModifyIndicationTransfer: &transfer,
				},
			},
		},
	}
}

func getPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId int64, pduSessionId int64, pduSessionResourceModifyIndicationTransferMessage []byte) ([]byte, error) {
	return buildPDUSessionResourceModifyIndication(amfUeNgapId, ranUeNgapId, pduSessionId, pduSessionResourceModifyIndicationTransferMessage).MarshalBinary()
}
