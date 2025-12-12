package util_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/free-ran-ue/free-ran-ue/model"
	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/free5gc/openapi/models"
	"github.com/stretchr/testify/assert"
)

var testValidateLoggerIeCases = []struct {
	name          string
	loggerIe      model.LoggerIE
	expectedError error
}{
	{
		name:          "testError",
		loggerIe:      model.LoggerIE{Level: "error"},
		expectedError: nil,
	},
	{
		name:          "testWarn",
		loggerIe:      model.LoggerIE{Level: "warn"},
		expectedError: nil,
	},
	{
		name:          "testInfo",
		loggerIe:      model.LoggerIE{Level: "info"},
		expectedError: nil,
	},
	{
		name:          "testDebug",
		loggerIe:      model.LoggerIE{Level: "debug"},
		expectedError: nil,
	},
	{
		name:          "testTrace",
		loggerIe:      model.LoggerIE{Level: "trace"},
		expectedError: nil,
	},
	{
		name:          "testTest",
		loggerIe:      model.LoggerIE{Level: "test"},
		expectedError: nil,
	},
	{
		name:          "testInvalid",
		loggerIe:      model.LoggerIE{Level: "invalid"},
		expectedError: fmt.Errorf("invalid logger level: invalid"),
	},
}

func TestValidateLoggerIe(t *testing.T) {
	for _, testCase := range testValidateLoggerIeCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateLoggerIe(&testCase.loggerIe)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateIpCases = []struct {
	name          string
	ip            string
	expectedError error
}{
	{
		name:          "testValidIp",
		ip:            "192.168.1.1",
		expectedError: nil,
	},
	{
		name:          "testInvalidRangeIp",
		ip:            "192.168.1.256",
		expectedError: fmt.Errorf("invalid ip address: 192.168.1.256"),
	},
	{
		name:          "testInvalidFormatIp",
		ip:            "192.168.1.1.1",
		expectedError: fmt.Errorf("invalid ip address: 192.168.1.1.1"),
	},
}

func TestValidateIp(t *testing.T) {
	for _, testCase := range testValidateIpCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateIp(testCase.ip)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidatePortCases = []struct {
	name          string
	port          int
	expectedError error
}{
	{
		name:          "testValidPort",
		port:          8080,
		expectedError: nil,
	},
	{
		name:          "testInvalidPort",
		port:          0,
		expectedError: fmt.Errorf("invalid port range: 0, range should be 1-65535"),
	},
}

func TestValidatePort(t *testing.T) {
	for _, testCase := range testValidatePortCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidatePort(testCase.port)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateIntStringWithLengthCases = []struct {
	name          string
	intString     string
	length        int
	expectedError error
}{
	{
		name:          "testValidIntString",
		intString:     "12345",
		length:        5,
		expectedError: nil,
	},
	{
		name:          "testInvalidIntString",
		intString:     "12345a",
		length:        5,
		expectedError: fmt.Errorf("invalid int string: 12345a"),
	},
	{
		name:          "testInvalidIntStringLength",
		intString:     "12345",
		length:        10,
		expectedError: fmt.Errorf("invalid int string: 12345, length should be 10"),
	},
}

func TestValidateIntString(t *testing.T) {
	for _, testCase := range testValidateIntStringWithLengthCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateIntStringWithLength(testCase.intString, testCase.length)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidatePlmnIdCases = []struct {
	name          string
	plmnId        model.PlmnIdIE
	expectedError error
}{
	{
		name:          "testValidPlmnId",
		plmnId:        model.PlmnIdIE{Mcc: "208", Mnc: "93"},
		expectedError: nil,
	},
	{
		name:          "testInvalidPlmnId",
		plmnId:        model.PlmnIdIE{Mcc: "208", Mnc: "930"},
		expectedError: fmt.Errorf("invalid mnc: 930, mnc should be 2 digits"),
	},
	{
		name:          "testInvalidNonIntMcc",
		plmnId:        model.PlmnIdIE{Mcc: "20a", Mnc: "93"},
		expectedError: fmt.Errorf("invalid int string: 20a"),
	},
	{
		name:          "testInvalidNonIntMnc",
		plmnId:        model.PlmnIdIE{Mcc: "208", Mnc: "9a"},
		expectedError: fmt.Errorf("invalid int string: 9a"),
	},
}

func TestValidatePlmnId(t *testing.T) {
	for _, testCase := range testValidatePlmnIdCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidatePlmnId(&testCase.plmnId)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateMsinCases = []struct {
	name          string
	msin          string
	expectedError error
}{
	{
		name:          "testValidMsin",
		msin:          "0000000001",
		expectedError: nil,
	},
	{
		name:          "testInvalidMsin",
		msin:          "00000000010",
		expectedError: fmt.Errorf("invalid msin: 00000000010, msin should be 10 digits"),
	},
	{
		name:          "testInvalidNonIntMsin",
		msin:          "000000000a",
		expectedError: fmt.Errorf("invalid int string: 000000000a"),
	},
}

func TestValidateMsin(t *testing.T) {
	for _, testCase := range testValidateMsinCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateMsin(testCase.msin)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateAccessTypeCases = []struct {
	name          string
	accessType    string
	expectedError error
}{
	{
		name:          "testValidAccessType",
		accessType:    "3GPP_ACCESS",
		expectedError: nil,
	},
	{
		name:          "testInvalidAccessType",
		accessType:    "INVALID",
		expectedError: fmt.Errorf("invalid access type: INVALID"),
	},
	{
		name:          "testUnsupportedAccessType",
		accessType:    "NON_3GPP_ACCESS",
		expectedError: fmt.Errorf("unsupported access type: NON_3GPP_ACCESS"),
	},
}

func TestValidateAccessType(t *testing.T) {
	for _, testCase := range testValidateAccessTypeCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateAccessType(models.AccessType(testCase.accessType))
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateAuthenticationSubscriptionCases = []struct {
	name                       string
	authenticationSubscription model.AuthenticationSubscriptionIE
	expectedError              error
}{
	{
		name: "testValidAuthenticationSubscription",
		authenticationSubscription: model.AuthenticationSubscriptionIE{
			EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
			EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
			AuthenticationManagementField: "8000",
			SequenceNumber:                "000000000023",
		},
		expectedError: nil,
	},
	{
		name: "testInvalidNonHexEncPermanentKey",
		authenticationSubscription: model.AuthenticationSubscriptionIE{
			EncPermanentKey:               "zzzzzzzzzzzz",
			EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
			AuthenticationManagementField: "8000",
			SequenceNumber:                "000000000023",
		},
		expectedError: fmt.Errorf("invalid enc permanent key, invalid hex string: zzzzzzzzzzzz"),
	},
	{
		name: "testInvalidNonHexEncOpcKey",
		authenticationSubscription: model.AuthenticationSubscriptionIE{
			EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
			EncOpcKey:                     "zzzzzzzzzzzz",
			AuthenticationManagementField: "8000",
			SequenceNumber:                "000000000023",
		},
		expectedError: fmt.Errorf("invalid enc opc key, invalid hex string: zzzzzzzzzzzz"),
	},
	{
		name: "testInvalidNonIntAuthenticationManagementField",
		authenticationSubscription: model.AuthenticationSubscriptionIE{
			EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
			EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
			AuthenticationManagementField: "800a",
			SequenceNumber:                "000000000023",
		},
		expectedError: fmt.Errorf("invalid authentication management field, invalid int string: 800a"),
	},
	{
		name: "testInvalidIntLengthAuthenticationManagementField",
		authenticationSubscription: model.AuthenticationSubscriptionIE{
			EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
			EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
			AuthenticationManagementField: "80000",
			SequenceNumber:                "000000000023",
		},
		expectedError: fmt.Errorf("invalid authentication management field, invalid int string: 80000, length should be 4"),
	},
}

func TestValidateAuthenticationSubscription(t *testing.T) {
	for _, testCase := range testValidateAuthenticationSubscriptionCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateAuthenticationSubscription(&testCase.authenticationSubscription)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateXorBooleanFlagCases = []struct {
	name          string
	booleanFlags  []bool
	expectedError error
}{
	{
		name:          "testValidXorBooleanFlag",
		booleanFlags:  []bool{true, false, false},
		expectedError: nil,
	},
	{
		name:          "testInvalidXorBooleanFlag",
		booleanFlags:  []bool{false, false, false},
		expectedError: fmt.Errorf("no true boolean flag, one true flag is required"),
	},
	{
		name:          "testInvalidXorBooleanFlag",
		booleanFlags:  []bool{true, true, false},
		expectedError: fmt.Errorf("exist multiple true boolean flags"),
	},
}

func TestValidateXorBooleanFlag(t *testing.T) {
	for _, testCase := range testValidateXorBooleanFlagCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateXorBooleanFlag(testCase.booleanFlags...)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateIntegrityAlgorithmCases = []struct {
	name               string
	integrityAlgorithm model.IntegrityAlgorithmIE
	expectedError      error
}{
	{
		name: "testValidIntegrityAlgorithm",
		integrityAlgorithm: model.IntegrityAlgorithmIE{
			Nia0: false,
			Nia1: false,
			Nia2: true,
			Nia3: false,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidMultipleTrueIntegrityAlgorithm",
		integrityAlgorithm: model.IntegrityAlgorithmIE{
			Nia0: false,
			Nia1: false,
			Nia2: true,
			Nia3: true,
		},
		expectedError: fmt.Errorf("exist multiple true boolean flags"),
	},
	{
		name: "testInvalidNoTrueIntegrityAlgorithm",
		integrityAlgorithm: model.IntegrityAlgorithmIE{
			Nia0: false,
			Nia1: false,
			Nia2: false,
			Nia3: false,
		},
		expectedError: fmt.Errorf("no true boolean flag, one true flag is required"),
	},
}

func TestValidateIntegrityAlgorithm(t *testing.T) {
	for _, testCase := range testValidateIntegrityAlgorithmCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateIntegrityAlgorithm(&testCase.integrityAlgorithm)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateCipheringAlgorithmCases = []struct {
	name               string
	cipheringAlgorithm model.CipheringAlgorithmIE
	expectedError      error
}{
	{
		name: "testValidCipheringAlgorithm",
		cipheringAlgorithm: model.CipheringAlgorithmIE{
			Nea0: true,
			Nea1: false,
			Nea2: false,
			Nea3: false,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidMultipleTrueCipheringAlgorithm",
		cipheringAlgorithm: model.CipheringAlgorithmIE{
			Nea0: true,
			Nea1: false,
			Nea2: false,
			Nea3: true,
		},
		expectedError: fmt.Errorf("exist multiple true boolean flags"),
	},
	{
		name: "testInvalidNoTrueCipheringAlgorithm",
		cipheringAlgorithm: model.CipheringAlgorithmIE{
			Nea0: false,
			Nea1: false,
			Nea2: false,
			Nea3: false,
		},
		expectedError: fmt.Errorf("no true boolean flag, one true flag is required"),
	},
}

func TestValidateCipheringAlgorithm(t *testing.T) {
	for _, testCase := range testValidateCipheringAlgorithmCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateCipheringAlgorithm(&testCase.cipheringAlgorithm)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidatePduSessionCases = []struct {
	name          string
	pduSession    model.PduSessionIE
	expectedError error
}{
	{
		name: "testValidPduSession",
		pduSession: model.PduSessionIE{
			Dnn: "internet",
			Snssai: model.SnssaiIE{
				Sst: "1",
				Sd:  "010203",
			},
		},
		expectedError: nil,
	},
	{
		name: "testInvalidSstNilPduSession",
		pduSession: model.PduSessionIE{
			Dnn: "internet",
			Snssai: model.SnssaiIE{
				Sst: "z",
				Sd:  "010203",
			},
		},
		expectedError: fmt.Errorf("invalid pdu session sst, invalid int string: z"),
	},
	{
		name: "testInvalidSdNilPduSession",
		pduSession: model.PduSessionIE{
			Dnn: "internet",
			Snssai: model.SnssaiIE{
				Sst: "1",
				Sd:  "zzzzzz",
			},
		},
		expectedError: fmt.Errorf("invalid pdu session sd, invalid hex string: zzzzzz"),
	},
}

func TestValidatePduSession(t *testing.T) {
	for _, testCase := range testValidatePduSessionCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidatePduSession(&testCase.pduSession)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateNrdcCases = []struct {
	name          string
	nrdc          model.NrdcIE
	expectedError error
}{
	{
		name: "testValidEnableNrdc",
		nrdc: model.NrdcIE{
			Enable: true,
			DcRanDataPlane: model.DcDataPlaneIE{
				Ip:   "10.0.3.1",
				Port: 31414,
			},
		},
		expectedError: nil,
	},
	{
		name: "testValidEnableNrdcWithLocalDataPlaneIp",
		nrdc: model.NrdcIE{
			Enable: true,
			DcRanDataPlane: model.DcDataPlaneIE{
				Ip:   "10.0.3.1",
				Port: 31414,
			},
			DcLocalDataPlaneIp: "10.0.3.2",
		},
		expectedError: nil,
	},
	{
		name: "testValidDisableNrdc",
		nrdc: model.NrdcIE{
			Enable: false,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidIpNrdc",
		nrdc: model.NrdcIE{
			Enable: true,
			DcRanDataPlane: model.DcDataPlaneIE{
				Ip:   "10.0.3.1.1",
				Port: 31414,
			},
		},
		expectedError: fmt.Errorf("invalid nrdc dc ran data plane ip, invalid ip address: 10.0.3.1.1"),
	},
	{
		name: "testInvalidPortNrdc",
		nrdc: model.NrdcIE{
			Enable: true,
			DcRanDataPlane: model.DcDataPlaneIE{
				Ip:   "10.0.3.1",
				Port: 0,
			},
		},
		expectedError: fmt.Errorf("invalid nrdc dc ran data plane port, invalid port range: 0, range should be 1-65535"),
	},
}

func TestValidateNrdc(t *testing.T) {
	for _, testCase := range testValidateNrdcCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateNrdc(&testCase.nrdc)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateUeIeCases = []struct {
	name          string
	ueIe          model.UeIE
	expectedError error
}{
	{
		name: "testValidUeIe",
		ueIe: model.UeIE{
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Msin:       "0000000001",
			AccessType: models.AccessType("3GPP_ACCESS"),
			AuthenticationSubscription: model.AuthenticationSubscriptionIE{
				EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
				EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
				AuthenticationManagementField: "8000",
				SequenceNumber:                "000000000023",
			},
			CipheringAlgorithm: model.CipheringAlgorithmIE{
				Nea0: true,
				Nea1: false,
				Nea2: false,
				Nea3: false,
			},
			IntegrityAlgorithm: model.IntegrityAlgorithmIE{
				Nia0: false,
				Nia1: false,
				Nia2: true,
				Nia3: false,
			},
			PduSession: model.PduSessionIE{
				Dnn: "internet",
				Snssai: model.SnssaiIE{
					Sst: "1",
					Sd:  "010203",
				},
			},
			UeTunnelDevice: "ueTun0",
		},
		expectedError: nil,
	},
	{
		name: "testValidStaticNrdcUeIe",
		ueIe: model.UeIE{
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			LocalDataPlaneIp:    "10.0.2.2",
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Msin:       "0000000001",
			AccessType: models.AccessType("3GPP_ACCESS"),
			AuthenticationSubscription: model.AuthenticationSubscriptionIE{
				EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
				EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
				AuthenticationManagementField: "8000",
				SequenceNumber:                "000000000023",
			},
			CipheringAlgorithm: model.CipheringAlgorithmIE{
				Nea0: true,
				Nea1: false,
				Nea2: false,
				Nea3: false,
			},
			IntegrityAlgorithm: model.IntegrityAlgorithmIE{
				Nia0: false,
				Nia1: false,
				Nia2: true,
				Nia3: false,
			},
			PduSession: model.PduSessionIE{
				Dnn: "internet",
				Snssai: model.SnssaiIE{
					Sst: "1",
					Sd:  "010203",
				},
			},
			Nrdc: model.NrdcIE{
				Enable: true,
				DcRanDataPlane: model.DcDataPlaneIE{
					Ip:   "10.0.3.1",
					Port: 31414,
				},
				DcLocalDataPlaneIp: "10.0.3.2",
			},
			UeTunnelDevice: "ueTun0",
		},
		expectedError: nil,
	},
	{
		name: "testValidDynamicNrdcUeIe",
		ueIe: model.UeIE{
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			LocalDataPlaneIp:    "10.0.2.2",
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Msin:       "0000000001",
			AccessType: models.AccessType("3GPP_ACCESS"),
			AuthenticationSubscription: model.AuthenticationSubscriptionIE{
				EncPermanentKey:               "8baf473f2f8fd09487cccbd7097c6862",
				EncOpcKey:                     "8e27b6af0e692e750f32667a3b14605d",
				AuthenticationManagementField: "8000",
				SequenceNumber:                "000000000023",
			},
			CipheringAlgorithm: model.CipheringAlgorithmIE{
				Nea0: true,
				Nea1: false,
				Nea2: false,
				Nea3: false,
			},
			IntegrityAlgorithm: model.IntegrityAlgorithmIE{
				Nia0: false,
				Nia1: false,
				Nia2: true,
				Nia3: false,
			},
			PduSession: model.PduSessionIE{
				Dnn: "internet",
				Snssai: model.SnssaiIE{
					Sst: "1",
					Sd:  "010203",
				},
			},
			Nrdc: model.NrdcIE{
				Enable: false,
				DcRanDataPlane: model.DcDataPlaneIE{
					Ip:   "10.0.3.1",
					Port: 31414,
				},
				DcLocalDataPlaneIp: "10.0.3.2",
			},
			UeTunnelDevice: "ueTun0",
		},
		expectedError: nil,
	},
}

func TestValidateUeIe(t *testing.T) {
	for _, testCase := range testValidateUeIeCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateUeIe(&testCase.ueIe)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

var testValidateTaiIeCases = []struct {
	name          string
	tai           model.TaiIE
	expectedError error
}{
	{
		name: "testValidTaiIe",
		tai: model.TaiIE{
			Tac: "000001",
			BroadcastPlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
		},
		expectedError: nil,
	},
	{
		name: "testInvalidTac",
		tai: model.TaiIE{
			Tac: "zzzzzzzz",
			BroadcastPlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
		},
		expectedError: fmt.Errorf("invalid tac: invalid hex string: zzzzzzzz"),
	},
	{
		name: "testInvalidBroadcastPlmnId",
		tai: model.TaiIE{
			Tac: "000001",
			BroadcastPlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "930",
			},
		},
		expectedError: fmt.Errorf("invalid broadcastPlmnId: invalid mnc: 930, mnc should be 2 digits"),
	},
}

func TestValidateTaiIe(t *testing.T) {
	for _, tc := range testValidateTaiIeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := util.ValidateTaiIe(&tc.tai)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testValidateSnssaiIeCases = []struct {
	name          string
	snssai        model.SnssaiIE
	expectedError error
}{
	{
		name: "testValidSnssaiIe",
		snssai: model.SnssaiIE{
			Sst: "1",
			Sd:  "010203",
		},
		expectedError: nil,
	},
	{
		name: "testInvalidSst",
		snssai: model.SnssaiIE{
			Sst: "z",
			Sd:  "010203",
		},
		expectedError: fmt.Errorf("invalid sst, invalid int string: z"),
	},
	{
		name: "testInvalidSd",
		snssai: model.SnssaiIE{
			Sst: "1",
			Sd:  "01020g",
		},
		expectedError: fmt.Errorf("invalid sd, invalid hex string: 01020g"),
	},
}

func TestValidateSnssaiIe(t *testing.T) {
	for _, tc := range testValidateSnssaiIeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := util.ValidateSnssaiIe(&tc.snssai)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testValidateApiIeCases = []struct {
	name          string
	api           model.ApiIE
	expectedError error
}{
	{
		name: "testValidApiIe",
		api: model.ApiIE{
			Ip:   "10.0.1.3",
			Port: 40104,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidIp",
		api: model.ApiIE{
			Ip:   "10.0.1.3.1",
			Port: 40104,
		},
		expectedError: fmt.Errorf("invalid ip: invalid ip address: 10.0.1.3.1"),
	},
	{
		name: "testInvalidPort",
		api: model.ApiIE{
			Ip:   "10.0.1.3",
			Port: 0,
		},
		expectedError: fmt.Errorf("invalid port: invalid port range: 0, range should be 1-65535"),
	},
}

func TestValidateApiIe(t *testing.T) {
	for _, tc := range testValidateApiIeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := util.ValidateApiIe(&tc.api)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testValidateXnInterfaceIeCases = []struct {
	name          string
	xn            model.XnInterfaceIE
	expectedError error
}{
	{
		name: "testValidXnInterfaceIe",
		xn: model.XnInterfaceIE{
			Enable:       true,
			XnListenIp:   "10.0.1.3",
			XnListenPort: 31415,
			XnDialIp:     "10.0.1.2",
			XnDialPort:   31415,
		},
		expectedError: nil,
	},
	{
		name: "testValidXnInterfaceIeDisabled",
		xn: model.XnInterfaceIE{
			Enable: false,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidXnListenIp",
		xn: model.XnInterfaceIE{
			Enable:       true,
			XnListenIp:   "10.0.1.3.1",
			XnListenPort: 31415,
			XnDialIp:     "10.0.1.2",
			XnDialPort:   31415,
		},
		expectedError: fmt.Errorf("invalid xnListenIp: invalid ip address: 10.0.1.3.1"),
	},
	{
		name: "testInvalidXnListenPort",
		xn: model.XnInterfaceIE{
			Enable:       true,
			XnListenIp:   "10.0.1.3",
			XnListenPort: 0,
			XnDialIp:     "10.0.1.2",
			XnDialPort:   31415,
		},
		expectedError: fmt.Errorf("invalid xnListenPort: invalid port range: 0, range should be 1-65535"),
	},
	{
		name: "testInvalidXnDialIp",
		xn: model.XnInterfaceIE{
			Enable:       true,
			XnListenIp:   "10.0.1.3",
			XnListenPort: 31415,
			XnDialIp:     "10.0.1.256",
			XnDialPort:   31415,
		},
		expectedError: fmt.Errorf("invalid xnDialIp: invalid ip address: 10.0.1.256"),
	},
	{
		name: "testInvalidXnDialPort",
		xn: model.XnInterfaceIE{
			Enable:       true,
			XnListenIp:   "10.0.1.3",
			XnListenPort: 31415,
			XnDialIp:     "10.0.1.2",
			XnDialPort:   0,
		},
		expectedError: fmt.Errorf("invalid xnDialPort: invalid port range: 0, range should be 1-65535"),
	},
}

func TestValidateXnInterfaceIe(t *testing.T) {
	for _, tc := range testValidateXnInterfaceIeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := util.ValidateXnInterfaceIe(&tc.xn)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testValidateGnbIeCases = []struct {
	name          string
	gnbIe         model.GnbIE
	expectedError error
}{
	{
		name: "testValidGnbIe",
		gnbIe: model.GnbIE{
			AmfN2Ip:             "10.0.1.1",
			RanN2Ip:             "10.0.1.2",
			UpfN3Ip:             "10.0.1.1",
			RanN3Ip:             "10.0.1.2",
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			AmfN2Port:           38412,
			RanN2Port:           38413,
			UpfN3Port:           2152,
			RanN3Port:           2152,
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			GnbId:               "000314",
			GnbName:             "gNB",
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Tai: model.TaiIE{
				Tac: "000001",
				BroadcastPlmnId: model.PlmnIdIE{
					Mcc: "208",
					Mnc: "93",
				},
			},
			Snssai: model.SnssaiIE{
				Sst: "1",
				Sd:  "010203",
			},
			Api: model.ApiIE{
				Ip:   "10.0.1.2",
				Port: 40104,
			},
		},
		expectedError: nil,
	},
	{
		name: "testDcStaticgNB",
		gnbIe: model.GnbIE{
			AmfN2Ip:             "10.0.1.1",
			RanN2Ip:             "10.0.1.2",
			UpfN3Ip:             "10.0.1.1",
			RanN3Ip:             "10.0.1.2",
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			AmfN2Port:           38412,
			RanN2Port:           38413,
			UpfN3Port:           2152,
			RanN3Port:           2152,
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			GnbId:               "000314",
			GnbName:             "gNB",
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Tai: model.TaiIE{
				Tac: "000001",
				BroadcastPlmnId: model.PlmnIdIE{
					Mcc: "208",
					Mnc: "93",
				},
			},
			Snssai: model.SnssaiIE{
				Sst: "1",
				Sd:  "010203",
			},
			StaticNrdc: true,
			XnInterface: model.XnInterfaceIE{
				Enable:       true,
				XnListenIp:   "10.0.1.2",
				XnListenPort: 31415,
				XnDialIp:     "10.0.1.3",
				XnDialPort:   31415,
			},
			Api: model.ApiIE{
				Ip:   "10.0.1.2",
				Port: 40104,
			},
		},
		expectedError: nil,
	},
	{
		name: "testDcDynamicgNB",
		gnbIe: model.GnbIE{
			AmfN2Ip:             "10.0.1.1",
			RanN2Ip:             "10.0.1.2",
			UpfN3Ip:             "10.0.1.1",
			RanN3Ip:             "10.0.1.2",
			RanControlPlaneIp:   "10.0.2.1",
			RanDataPlaneIp:      "10.0.2.1",
			AmfN2Port:           38412,
			RanN2Port:           38413,
			UpfN3Port:           2152,
			RanN3Port:           2152,
			RanControlPlanePort: 31413,
			RanDataPlanePort:    31414,
			GnbId:               "000314",
			GnbName:             "gNB",
			PlmnId: model.PlmnIdIE{
				Mcc: "208",
				Mnc: "93",
			},
			Tai: model.TaiIE{
				Tac: "000001",
				BroadcastPlmnId: model.PlmnIdIE{
					Mcc: "208",
					Mnc: "93",
				},
			},
			Snssai: model.SnssaiIE{
				Sst: "1",
				Sd:  "010203",
			},
			StaticNrdc: false,
			XnInterface: model.XnInterfaceIE{
				Enable:       true,
				XnListenIp:   "10.0.1.2",
				XnListenPort: 31415,
				XnDialIp:     "10.0.1.3",
				XnDialPort:   31415,
			},
			Api: model.ApiIE{
				Ip:   "10.0.1.2",
				Port: 40104,
			},
		},
		expectedError: nil,
	},
}

func TestValidateGnbIe(t *testing.T) {
	for _, testCase := range testValidateGnbIeCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateGnbIe(&testCase.gnbIe)
			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testJWTIECases = []struct {
	name          string
	jwtIe         model.JWTIE
	expectedError error
}{
	{
		name: "testValidJWTIE",
		jwtIe: model.JWTIE{
			Secret:    "testSecret",
			ExpiresIn: 1 * time.Hour,
		},
		expectedError: nil,
	},
	{
		name: "testInvalidExpiresIn",
		jwtIe: model.JWTIE{
			Secret:    "testSecret",
			ExpiresIn: 0,
		},
		expectedError: fmt.Errorf("invalid expiresIn: 0, expiresIn must be positive"),
	},
}

func TestValidateJWTIE(t *testing.T) {
	for _, testCase := range testJWTIECases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateJWTIE(&testCase.jwtIe)
			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testFrontendFilePathCases = []struct {
	name             string
	frontendFilePath string
	expectedError    error
}{
	{
		name:             "testValidFrontendFilePath",
		frontendFilePath: "frontend",
		expectedError:    nil,
	},
	{
		name:             "testInvalidFrontendFilePath",
		frontendFilePath: "frontend/invalid",
		expectedError:    fmt.Errorf("no such file or directory"),
	},
}

func TestValidateFrontendFilePath(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "frontend")
	if err := os.Mkdir(validPath, 0755); err != nil {
		t.Fatalf("Failed to create valid path: %v", err)
	}

	for i := range testFrontendFilePathCases {
		testFrontendFilePathCases[i].frontendFilePath = filepath.Join(dir, testFrontendFilePathCases[i].frontendFilePath)
	}

	for _, testCase := range testFrontendFilePathCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateFrontendFilePath(testCase.frontendFilePath)
			if testCase.expectedError != nil {
				assert.True(t, strings.Contains(err.Error(), testCase.expectedError.Error()))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var testConsoleIeCases = []struct {
	name          string
	consoleIe     model.ConsoleIE
	expectedError error
}{
	{
		name: "testValidConsoleIe",
		consoleIe: model.ConsoleIE{
			Username: "admin",
			Password: "free-ran-ue",
			Port:     40104,
			JWT: model.JWTIE{
				Secret:    "free-ran-ue",
				ExpiresIn: 1 * time.Hour,
			},
			FrontendFilePath: "",
		},
		expectedError: nil,
	},
}

func TestValidateConsoleIe(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "frontend")
	if err := os.Mkdir(validPath, 0755); err != nil {
		t.Fatalf("Failed to create valid path: %v", err)
	}

	for i := range testConsoleIeCases {
		testConsoleIeCases[i].consoleIe.FrontendFilePath = filepath.Join(dir, testConsoleIeCases[i].consoleIe.FrontendFilePath)
	}

	for _, testCase := range testConsoleIeCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := util.ValidateConsoleIe(&testCase.consoleIe)
			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
