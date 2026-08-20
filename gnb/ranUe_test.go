package gnb

import (
	"testing"

	"github.com/free5gc/nas/ie"
)

var testGetMobileIdentityIMSICases = []struct {
	name     string
	buffer   []byte
	expected string
}{
	{
		name:     "3-digit-mnc-939",
		buffer:   []byte{0x01, 0x02, 0x98, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf1},
		expected: "imsi-208939000000001",
	},
	{
		name:     "2-digit-mnc-93",
		buffer:   []byte{0x01, 0x02, 0xf8, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10},
		expected: "imsi-208930000000001",
	},
}

func TestGetMobileIdentityIMSI(t *testing.T) {
	for _, tc := range testGetMobileIdentityIMSICases {
		t.Run(tc.name, func(t *testing.T) {
			mobileIdentity5GS := new(ie.MobileId5GS)
			if err := mobileIdentity5GS.UnmarshalBinary(tc.buffer); err != nil {
				t.Fatalf("unexpected error unmarshaling mobile identity: %v", err)
			}

			r := &RanUe{mobileIdentity5GS: mobileIdentity5GS}

			if got := r.GetMobileIdentityIMSI(); got != tc.expected {
				t.Fatalf("unexpected IMSI: got %s, want %s", got, tc.expected)
			}
		})
	}
}
