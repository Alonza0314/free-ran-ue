package ue

import (
	"testing"

	"github.com/free-ran-ue/util"
	"github.com/free5gc/nas/ie"
	"github.com/free5gc/openapi/models"
	"github.com/go-playground/assert"
)

var testBuildUeMobileIdentity5GSCases = []struct {
	name      string
	mccLength int
	mncLength int
	supi      string
}{
	{
		name:      "imsi-2089300007487",
		mccLength: 3,
		mncLength: 2,
		supi:      "2089300007487",
	},
	{
		name:      "imsi-208930000000001",
		mccLength: 3,
		mncLength: 2,
		supi:      "208930000000001",
	},
	{
		name:      "imsi-001001000000001",
		mccLength: 3,
		mncLength: 3,
		supi:      "001001000000001",
	},
	{
		name:      "imsi-208939000000001",
		mccLength: 3,
		mncLength: 3,
		supi:      "208939000000001",
	},
}

func TestBuildUeMobileIdentity5GS(t *testing.T) {
	for _, testCase := range testBuildUeMobileIdentity5GSCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := buildUeMobileIdentity5GS(testCase.mccLength, testCase.mncLength, testCase.supi)
			assert.Equal(t, nil, err)

			expected := util.SupiToBytes(testCase.mccLength, testCase.mncLength, testCase.supi)
			actual, err := result.MarshalBinary()
			assert.Equal(t, nil, err)
			assert.Equal(t, expected, actual)
		})
	}
}

var testBuildUeRegistrationRequestCases = []struct {
	name          string
	mccLength     int
	mncLength     int
	supi          string
	expectedError error
}{
	{
		name:          "imsi-208930000007487",
		mccLength:     3,
		mncLength:     2,
		supi:          "208930000007487",
		expectedError: nil,
	},
}

func TestBuildUeRegistrationRequest(t *testing.T) {
	for _, testCase := range testBuildUeRegistrationRequestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mobileIdentity5GS, err := buildUeMobileIdentity5GS(testCase.mccLength, testCase.mncLength, testCase.supi)
			assert.Equal(t, nil, err)

			result, err := buildUeRegistrationRequest(ie.RegType_InitialReg, mobileIdentity5GS, nil, nil, nil, nil, nil)
			assert.Equal(t, testCase.expectedError, err)
			assert.NotEqual(t, nil, result)
		})
	}
}

var testBuildAuthenticationResponseCases = []struct {
	name          string
	param         []byte
	expectedError error
}{
	{
		name:          "testBuildAuthenticationResponse",
		param:         []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		expectedError: nil,
	},
}

func TestBuildAuthenticationResponse(t *testing.T) {
	for _, testCase := range testBuildAuthenticationResponseCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := buildAuthenticationResponse(testCase.param)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testBuildNasSecurityModeCompleteMessageCases = []struct {
	name  string
	param []byte
}{
	{
		name:  "testBuildNasSecurityModeCompleteMessage",
		param: []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	},
}

func TestBuildNasSecurityModeCompleteMessage(t *testing.T) {
	for _, testCase := range testBuildNasSecurityModeCompleteMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildNasSecurityModeCompleteMessage(testCase.param)
			assert.NotEqual(t, nil, result)
			_, err := result.MarshalBinary()
			assert.Equal(t, nil, err)
		})
	}
}

func TestBuildNasRegistrationCompleteMessage(t *testing.T) {
	result := buildNasRegistrationCompleteMessage()
	assert.NotEqual(t, nil, result)
	_, err := result.MarshalBinary()
	assert.Equal(t, nil, err)
}

var testBuildPduSessionEstablishmentRequestCases = []struct {
	name          string
	pduSessionId  uint8
	expectedError error
}{
	{
		name:          "testBuildPduSessionEstablishmentRequest",
		pduSessionId:  4,
		expectedError: nil,
	},
}

func TestBuildPduSessionEstablishmentRequest(t *testing.T) {
	for _, testCase := range testBuildPduSessionEstablishmentRequestCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := buildPduSessionEstablishmentRequest(testCase.pduSessionId)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testBuildUlNasTransportMessageCases = []struct {
	name                string
	nasMessageContainer []byte
	pduSessionId        uint8
	requestType         ie.ConstReqType
	dnn                 string
	sNssai              *models.Snssai
}{
	{
		name:                "testBuildUlNasTransportMessage",
		nasMessageContainer: []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		pduSessionId:        4,
		requestType:         ie.ReqType_InitialReq,
		dnn:                 "internet",
		sNssai: &models.Snssai{
			Sst: 1,
			Sd:  "010203",
		},
	},
	{
		name:                "testBuildUlNasTransportMessageWithoutSD",
		nasMessageContainer: []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		pduSessionId:        4,
		requestType:         ie.ReqType_InitialReq,
		dnn:                 "internet",
		sNssai: &models.Snssai{
			Sst: 1,
			Sd:  "",
		},
	},
}

func TestBuildUlNasTransportMessage(t *testing.T) {
	for _, testCase := range testBuildUlNasTransportMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildUlNasTransportMessage(testCase.nasMessageContainer, testCase.pduSessionId, testCase.requestType, testCase.dnn, testCase.sNssai)
			assert.NotEqual(t, nil, result)
			_, err := result.MarshalBinary()
			assert.Equal(t, nil, err)
		})
	}
}

var testBuildUeDeRegistrationRequestCases = []struct {
	name       string
	accessType uint8
	switchOff  uint8
	ngKsi      uint8
	mccLength  int
	mncLength  int
	supi       string
}{
	{
		name:       "imsi-208930000007487",
		accessType: ie.AccessType_3gpp,
		switchOff:  0x00,
		ngKsi:      ie.NASKeyNA,
		mccLength:  3,
		mncLength:  2,
		supi:       "208930000007487",
	},
}

func TestBuildUeDeRegistrationRequest(t *testing.T) {
	for _, testCase := range testBuildUeDeRegistrationRequestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mobileIdentity5GS, err := buildUeMobileIdentity5GS(testCase.mccLength, testCase.mncLength, testCase.supi)
			assert.Equal(t, nil, err)

			result := buildUeDeRegistrationRequest(testCase.accessType, testCase.switchOff, testCase.ngKsi, mobileIdentity5GS)
			assert.NotEqual(t, nil, result)
			_, err = result.MarshalBinary()
			assert.Equal(t, nil, err)
		})
	}
}
