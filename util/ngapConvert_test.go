package util_test

import (
	"testing"

	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/go-playground/assert/v2"
)

var testPlmnIdCases = []struct {
	name         string
	modelsPlmnId models.PlmnId
	ngapPlmnId   ngapType.PLMNIdentity
}{
	{
		name: "testPlmnId",
		modelsPlmnId: models.PlmnId{
			Mcc: "208",
			Mnc: "93",
		},
		ngapPlmnId: ngapType.PLMNIdentity{
			Value: []byte{0x02, 0xF8, 0x39},
		},
	},
}

func TestPlmnIdToModels(t *testing.T) {
	for _, testCase := range testPlmnIdCases {
		t.Run(testCase.name, func(t *testing.T) {
			modelsPlmnId := util.PlmnIdToModels(testCase.ngapPlmnId)
			assert.Equal(t, testCase.modelsPlmnId, modelsPlmnId)
			ngapPlmnId, err := util.PlmnIdToNgap(testCase.modelsPlmnId)
			assert.Equal(t, testCase.ngapPlmnId, ngapPlmnId)
			assert.Equal(t, nil, err)
		})
	}
}

var testTaiCases = []struct {
	name      string
	modelsTai models.Tai
	ngapTai   ngapType.TAI
}{
	{
		name: "testTai",
		modelsTai: models.Tai{
			PlmnId: &models.PlmnId{
				Mcc: "208",
				Mnc: "93",
			},
			Tac: "000001",
		},
		ngapTai: ngapType.TAI{
			PLMNIdentity: ngapType.PLMNIdentity{
				Value: []byte{0x02, 0xF8, 0x39},
			},
			TAC: ngapType.TAC{
				Value: []byte{0x00, 0x00, 0x01},
			},
		},
	},
}

func TestTaiToModels(t *testing.T) {
	for _, testCase := range testTaiCases {
		t.Run(testCase.name, func(t *testing.T) {
			modelsTai := util.TaiToModels(testCase.ngapTai)
			assert.Equal(t, testCase.modelsTai, modelsTai)
			ngapTai, err := util.TaiToNgap(testCase.modelsTai)
			assert.Equal(t, testCase.ngapTai, ngapTai)
			assert.Equal(t, nil, err)
		})
	}
}

var testSnssaiCases = []struct {
	name         string
	modelsSnssai models.Snssai
	ngapSnssai   ngapType.SNSSAI
}{
	{
		name: "testSnssai",
		modelsSnssai: models.Snssai{
			Sst: 1,
			Sd:  "010203",
		},
		ngapSnssai: ngapType.SNSSAI{
			SST: ngapType.SST{
				Value: []byte{0x01},
			},
			SD: &ngapType.SD{
				Value: []byte{0x01, 0x02, 0x03},
			},
		},
	},
}

func TestSnssaiToModels(t *testing.T) {
	for _, testCase := range testSnssaiCases {
		t.Run(testCase.name, func(t *testing.T) {
			modelsSnssai := util.SNssaiToModels(testCase.ngapSnssai)
			assert.Equal(t, testCase.modelsSnssai, modelsSnssai)
			ngapSnssai, err := util.SNssaiToNgap(testCase.modelsSnssai)
			assert.Equal(t, testCase.ngapSnssai, ngapSnssai)
			assert.Equal(t, nil, err)
		})
	}
}
