package util_test

import (
	"testing"

	"github.com/Alonza0314/free-ran-ue/util"
	"github.com/go-playground/assert/v2"
)

var testCases = []struct {
	name      string
	rawPacket []byte
	qosFlow   []string
	expected  bool
}{
	{
		name: "ip in qos flow",
		rawPacket: []byte{
			0x45, 0x00, 0x00, 0x73, 0x00,
			0x00, 0x40, 0x00, 0x40, 0x11,
			0x00, 0x00, 0x7f, 0x00, 0x00,
			0x01, 0x01, 0x01, 0x01, 0x01,
		},
		qosFlow:  []string{"1.1.1.1/32"},
		expected: true,
	},
	{
		name: "ip in qos flow",
		rawPacket: []byte{
			0x45, 0x00, 0x00, 0x73, 0x00,
			0x00, 0x40, 0x00, 0x40, 0x11,
			0x00, 0x00, 0x7f, 0x00, 0x00,
			0x01, 0x01, 0x01, 0x01, 0x01,
		},
		qosFlow:  []string{"1.1.1.2/24"},
		expected: true,
	},
	{
		name: "ip in not qos flow",
		rawPacket: []byte{
			0x45, 0x00, 0x00, 0x73, 0x00,
			0x00, 0x40, 0x00, 0x40, 0x11,
			0x00, 0x00, 0x7f, 0x00, 0x00,
			0x01, 0x08, 0x08, 0x08, 0x08,
		},
		qosFlow:  []string{"1.1.1.1/32"},
		expected: false,
	},
}

func TestIsIpInQosFlow(t *testing.T) {
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := util.IsIpInQosFlow(testCase.rawPacket, testCase.qosFlow)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
