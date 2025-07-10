package ue

import (
	"testing"

	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/go-playground/assert"
)

var testBuildUeMobileIdentity5GSCases = []struct {
	name     string
	supi     string
	expected nasType.MobileIdentity5GS
}{
	{
		name: "imsi-2089300007487",
		supi: "2089300007487",
		expected: nasType.MobileIdentity5GS{
			Len:    13,
			Buffer: []byte{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		},
	},
	{
		name: "imsi-208930000000001",
		supi: "208930000000001",
		expected: nasType.MobileIdentity5GS{
			Len:    13,
			Buffer: []byte{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10},
		},
	},
}

func TestBuildUeMobileIdentity5GS(t *testing.T) {
	for _, testCase := range testBuildUeMobileIdentity5GSCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildUeMobileIdentity5GS(testCase.supi)
			assert.Equal(t, testCase.expected.Len, result.Len)
			assert.Equal(t, testCase.expected.Buffer, result.Buffer)
		})
	}
}

var testBuildUeRegistrationRequestCases = []struct {
	name              string
	mobileIdentity5GS nasType.MobileIdentity5GS
	expectedError     error
	expected          []byte
}{
	{
		name: "imsi-208930000007487",
		mobileIdentity5GS: nasType.MobileIdentity5GS{
			Len:    12,
			Buffer: []byte{0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		},
		expectedError: nil,
		expected:      []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	},
}

func TestBuildUeRegistrationRequest(t *testing.T) {
	for _, testCase := range testBuildUeRegistrationRequestCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := buildUeRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration, &testCase.mobileIdentity5GS, nil, nil, nil, nil, nil)
			assert.Equal(t, testCase.expectedError, err)
			assert.Equal(t, testCase.expected, result)
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
	name          string
	param         []byte
	expectedError error
}{
	{
		name:          "testBuildNasSecurityModeCompleteMessage",
		param:         []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		expectedError: nil,
	},
}

func TestBuildNasSecurityModeCompleteMessage(t *testing.T) {
	for _, testCase := range testBuildNasSecurityModeCompleteMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := buildNasSecurityModeCompleteMessage(testCase.param)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testBuildNasRegistrationCompleteMessageCases = []struct {
	name          string
	param         []byte
	expectedError error
}{
	{
		name:          "testBuildNasRegistrationCompleteMessage",
		param:         []byte{0x7e, 0x00, 0x41, 0x79, 0x00, 0x0c, 0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
		expectedError: nil,
	},
}

func TestBuildNasRegistrationCompleteMessage(t *testing.T) {
	for _, testCase := range testBuildNasRegistrationCompleteMessageCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := buildNasRegistrationCompleteMessage(testCase.param)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}
