package gnb

import (
	"reflect"
	"testing"

	"github.com/free5gc/ngap/aper"
	"github.com/free5gc/ngap/ie"
	"github.com/free5gc/ngap/message"
	"github.com/stretchr/testify/assert"
)

var testBuildNgapSetupRequestCases = []struct {
	name    string
	gnbId   []byte
	gnbName string
	plmnId  ie.PLMNIdentity
	tai     ie.TAI
	snssai  ie.SNSSAI
}{
	{
		name:    "testBuildNgapSetupRequest",
		gnbId:   []byte("\x00\x03\x14"),
		gnbName: "gNB",
		plmnId: ie.PLMNIdentity{
			Value: aper.OctetString("\x02\xF8\x39"),
		},
		tai: ie.TAI{
			TAC: &ie.TAC{
				Value: aper.OctetString("\x00\x00\x01"),
			},
			PLMNIdentity: &ie.PLMNIdentity{
				Value: aper.OctetString("\x02\xF8\x39"),
			},
		},
		snssai: ie.SNSSAI{
			SST: &ie.SST{
				Value: aper.OctetString("\x01"),
			},
			SD: &ie.SD{
				Value: aper.OctetString("\x01\x02\x03"),
			},
		},
	},
	{
		name:    "testBuildNgapSetupRequestWithoutSD",
		gnbId:   []byte("\x00\x03\x14"),
		gnbName: "gNB",
		plmnId: ie.PLMNIdentity{
			Value: aper.OctetString("\x02\xF8\x39"),
		},
		tai: ie.TAI{
			TAC: &ie.TAC{
				Value: aper.OctetString("\x00\x00\x01"),
			},
			PLMNIdentity: &ie.PLMNIdentity{
				Value: aper.OctetString("\x02\xF8\x39"),
			},
		},
		snssai: ie.SNSSAI{
			SST: &ie.SST{
				Value: aper.OctetString("\x01"),
			},
		},
	},
}

func TestBuildNgapSetupRequest(t *testing.T) {
	for _, testCase := range testBuildNgapSetupRequestCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildNgapSetupRequest(testCase.gnbId, testCase.gnbName, testCase.plmnId, testCase.tai, testCase.snssai)
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP setup request: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP setup request: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.NGSetupRequest)
			if !ok {
				t.Fatalf("Decoded message is not NGSetupRequest: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP setup request mismatch")
			}
		})
	}
}

var testBuildIntialUeMessageCases = []struct {
	name                  string
	ranUeNgapId           int64
	ueRegistrationRequest []byte
	plmnId                ie.PLMNIdentity
	tai                   ie.TAI
}{
	{
		name:                  "testBuildIntialUeMessage",
		ranUeNgapId:           1,
		ueRegistrationRequest: []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		plmnId: ie.PLMNIdentity{
			Value: aper.OctetString("\x02\xF8\x39"),
		},
		tai: ie.TAI{
			TAC: &ie.TAC{
				Value: aper.OctetString("\x00\x00\x01"),
			},
			PLMNIdentity: &ie.PLMNIdentity{
				Value: aper.OctetString("\x02\xF8\x39"),
			},
		},
	},
}

func TestBuildIntialUeMessage(t *testing.T) {
	for _, testCase := range testBuildIntialUeMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildInitialUeMessage(testCase.ranUeNgapId, testCase.ueRegistrationRequest, testCase.plmnId, testCase.tai, []byte{0x00, 0x03, 0x14})
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP initial ue message: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP initial ue message: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.InitialUEMessage)
			if !ok {
				t.Fatalf("Decoded message is not InitialUEMessage: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP initial ue message mismatch")
			}
		})
	}
}

var testBuildUplinkNasTransportCases = []struct {
	name        string
	amfUeNgapId int64
	ranUeNgapId int64
	plmnId      ie.PLMNIdentity
	tai         ie.TAI
	nasPdu      []byte
}{
	{
		name:        "testBuildUplinkNasTransport",
		amfUeNgapId: 1,
		ranUeNgapId: 1,
		plmnId: ie.PLMNIdentity{
			Value: aper.OctetString("\x02\xF8\x39"),
		},
		tai: ie.TAI{
			TAC: &ie.TAC{
				Value: aper.OctetString("\x00\x00\x01"),
			},
			PLMNIdentity: &ie.PLMNIdentity{
				Value: aper.OctetString("\x02\xF8\x39"),
			},
		},
		nasPdu: []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
	},
}

func TestBuildUplinkNasTransport(t *testing.T) {
	for _, testCase := range testBuildUplinkNasTransportCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildUplinkNasTransport(testCase.amfUeNgapId, testCase.ranUeNgapId, testCase.plmnId, testCase.tai, testCase.nasPdu, []byte{0x00, 0x03, 0x14})
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP uplink nas transport: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP uplink nas transport: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.UplinkNASTransport)
			if !ok {
				t.Fatalf("Decoded message is not UplinkNASTransport: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP uplink nas transport mismatch")
			}
		})
	}
}

var testBuildNgapInitialContextSetupResponseCases = []struct {
	name        string
	amfUeNgapId int64
	ranUeNgapId int64
}{
	{
		name:        "testBuildNgapInitialContextSetupResponse",
		amfUeNgapId: 1,
		ranUeNgapId: 1,
	},
}

func TestBuildNgapInitialContextSetupResponse(t *testing.T) {
	for _, testCase := range testBuildNgapInitialContextSetupResponseCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildNgapInitialContextSetupResponse(testCase.amfUeNgapId, testCase.ranUeNgapId)
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP initial context setup response: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP initial context setup response: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.InitialContextSetupResponse)
			if !ok {
				t.Fatalf("Decoded message is not InitialContextSetupResponse: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP initial context setup response mismatch")
			}
		})
	}
}

func TestBuildNRCellIdentityFromGnbID(t *testing.T) {
	nrCellIdentity := buildNRCellIdentityFromGnbID([]byte{0x00, 0x03, 0x14})

	assert.Equal(t, uint64(36), nrCellIdentity.BitLength)
	assert.Equal(t, []byte{0x00, 0x03, 0x14, 0x00, 0x10}, nrCellIdentity.Bytes)
}

var testBuildPduSessionResourceSetupResponseTransferMessageCases = []struct {
	name    string
	dlTeid  []byte
	ranN3Ip string
	qosId   int64
}{
	{
		name:    "testBuildPduSessionResourceSetupResponseTransferMessage",
		dlTeid:  []byte("\x00\x00\x00\x01"),
		ranN3Ip: "127.0.0.1",
		qosId:   1,
	},
}

func TestBuildPduSessionResourceSetupResponseTransferMessage(t *testing.T) {
	for _, testCase := range testBuildPduSessionResourceSetupResponseTransferMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			transferMessage := buildPduSessionResourceSetupResponseTransfer(testCase.dlTeid, testCase.ranN3Ip, testCase.qosId, false, ie.QosFlowPerTNLInformationItem{})
			encodeTransferMessage, err := ie.MarshalBinary(&transferMessage)
			if err != nil {
				t.Fatalf("Failed to marshal pdu session resource setup response transfer message: %v", err)
			}

			decodeTransferMessage := &ie.PDUSessionResourceSetupResponseTransfer{}
			if err := ie.UnmarshalBinary(encodeTransferMessage, decodeTransferMessage); err != nil {
				t.Fatalf("Failed to unmarshal pdu session resource setup response transfer message: %v", err)
			}
			if !reflect.DeepEqual(&transferMessage, decodeTransferMessage) {
				t.Fatalf("PDU session resource setup response transfer message mismatch")
			}
		})
	}
}

var testBuildPduSessionResourceSetupResponseTransferMessageWithNRDCases = []struct {
	name    string
	dlTeid  []byte
	ranN3Ip string
	qosId   int64
	ie.QosFlowPerTNLInformationItem
}{
	{
		name:    "testBuildPduSessionResourceSetupResponseTransferMessageWithNRDCases",
		dlTeid:  []byte("\x00\x00\x00\x01"),
		ranN3Ip: "127.0.0.1",
		qosId:   1,
		QosFlowPerTNLInformationItem: ie.QosFlowPerTNLInformationItem{
			QosFlowPerTNLInformation: &ie.QosFlowPerTNLInformation{
				UPTransportLayerInformation: &ie.UPTransportLayerInformation{
					Choice: &ie.GTPTunnel{
						GTPTEID: &ie.GTPTEID{
							Value: aper.OctetString("\x00\x00\x00\x01"),
						},
						TransportLayerAddress: &ie.TransportLayerAddress{
							Value: ngapConvertIPAddressToNgap("127.0.0.1"),
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
		},
	},
}

func TestBuildPduSessionResourceSetupResponseTransferMessageWithNRDCases(t *testing.T) {
	for _, testCase := range testBuildPduSessionResourceSetupResponseTransferMessageWithNRDCases {
		t.Run(testCase.name, func(t *testing.T) {
			transferMessage := buildPduSessionResourceSetupResponseTransfer(testCase.dlTeid, testCase.ranN3Ip, testCase.qosId, true, testCase.QosFlowPerTNLInformationItem)
			encodeTransferMessage, err := ie.MarshalBinary(&transferMessage)
			if err != nil {
				t.Fatalf("Failed to marshal pdu session resource setup response transfer message: %v", err)
			}

			decodeTransferMessage := &ie.PDUSessionResourceSetupResponseTransfer{}
			if err := ie.UnmarshalBinary(encodeTransferMessage, decodeTransferMessage); err != nil {
				t.Fatalf("Failed to unmarshal pdu session resource setup response transfer message: %v", err)
			}
			if !reflect.DeepEqual(&transferMessage, decodeTransferMessage) {
				t.Fatalf("PDU session resource setup response transfer message mismatch")
			}
		})
	}
}

var testBuildPduSessionResourceSetupResponseCases = []struct {
	name            string
	amfUeNgapId     int64
	ranUeNgapId     int64
	pduSessionId    int64
	transferMessage []byte
}{
	{
		name:            "testBuildPduSessionResourceSetupResponse",
		amfUeNgapId:     1,
		ranUeNgapId:     1,
		pduSessionId:    1,
		transferMessage: []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
	},
}

func TestBuildPduSessionResourceSetupResponse(t *testing.T) {
	for _, testCase := range testBuildPduSessionResourceSetupResponseCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildPduSessionResourceSetupResponse(testCase.amfUeNgapId, testCase.ranUeNgapId, testCase.pduSessionId, testCase.transferMessage)
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP pdu session resource setup response: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP pdu session resource setup response: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.PDUSessionResourceSetupResponse)
			if !ok {
				t.Fatalf("Decoded message is not PDUSessionResourceSetupResponse: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP pdu session resource setup response mismatch")
			}
		})
	}
}

var testBuildNgapUeContextReleaseCompleteMessageCases = []struct {
	name             string
	amfUeNgapId      int64
	ranUeNgapId      int64
	pduSessionIdList []int64
	plmnId           ie.PLMNIdentity
	tai              ie.TAI
}{
	{
		name:             "testBuildNgapUeContextReleaseCommand",
		amfUeNgapId:      1,
		ranUeNgapId:      1,
		pduSessionIdList: []int64{1},
		plmnId: ie.PLMNIdentity{
			Value: aper.OctetString("\x02\xF8\x39"),
		},
		tai: ie.TAI{
			TAC: &ie.TAC{
				Value: aper.OctetString("\x00\x00\x01"),
			},
			PLMNIdentity: &ie.PLMNIdentity{
				Value: aper.OctetString("\x02\xF8\x39"),
			},
		},
	},
}

func TestBuildNgapUeContextReleaseCompleteMessage(t *testing.T) {
	for _, testCase := range testBuildNgapUeContextReleaseCompleteMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			pdu := buildNgapUeContextReleaseCompleteMessage(testCase.amfUeNgapId, testCase.ranUeNgapId, testCase.pduSessionIdList, testCase.plmnId, testCase.tai, []byte{0x00, 0x03, 0x14})
			encodeData, err := pdu.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to encode NGAP ue context release command: %v", err)
			}

			decodeMsg, err := message.Parse(encodeData)
			if err != nil {
				t.Fatalf("Failed to decode NGAP ue context release command: %v", err)
			}
			decodeData, ok := decodeMsg.(*message.UEContextReleaseComplete)
			if !ok {
				t.Fatalf("Decoded message is not UEContextReleaseComplete: %T", decodeMsg)
			}
			if !reflect.DeepEqual(pdu, decodeData) {
				t.Fatalf("NGAP ue context release command mismatch")
			}
		})
	}
}

var testBuildPDUSessionResourceModifyIndicationTransferCases = []struct {
	name    string
	dlTeid  []byte
	ranN3Ip string
	qosId   int64
}{
	{
		name:    "testBuildPDUSessionResourceModifyIndicationTransfer",
		dlTeid:  []byte("\x00\x00\x00\x01"),
		ranN3Ip: "127.0.0.1",
		qosId:   1,
	},
}

func TestBuildPDUSessionResourceModifyIndicationTransfer(t *testing.T) {
	for _, testCase := range testBuildPDUSessionResourceModifyIndicationTransferCases {
		t.Run(testCase.name, func(t *testing.T) {
			transferMessage := buildPDUSessionResourceModifyIndicationTransfer(testCase.dlTeid, testCase.ranN3Ip, testCase.qosId)
			encodeTransferMessage, err := ie.MarshalBinary(&transferMessage)
			if err != nil {
				t.Fatalf("Failed to marshal pdu session resource modify indication transfer message: %v", err)
			}

			decodeTransferMessage := &ie.PDUSessionResourceModifyIndicationTransfer{}
			if err := ie.UnmarshalBinary(encodeTransferMessage, decodeTransferMessage); err != nil {
				t.Fatalf("Failed to unmarshal pdu session resource modify indication transfer message: %v", err)
			}
			if !reflect.DeepEqual(&transferMessage, decodeTransferMessage) {
				t.Fatalf("PDU session resource modify indication transfer message mismatch")
			}
		})
	}
}
