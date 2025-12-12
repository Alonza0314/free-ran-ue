package util_test

import (
	"errors"
	"net"
	"testing"

	"github.com/free-ran-ue/free-ran-ue/util"
)

var testUdpDialWithOptionalLocalAddressCases = []struct {
	name          string
	remoteAddress string
	remotePort    int
	localAddress  string
}{
	{
		name:          "testUdpDialWithoutLocalAddress",
		remoteAddress: "127.0.0.1",
		remotePort:    8080,
		localAddress:  "",
	},
	{
		name:          "testUdpDialWithLocalAddress",
		remoteAddress: "127.0.0.1",
		remotePort:    8080,
		localAddress:  "127.0.0.1",
	},
}

func TestUdpDialWithOptionalLocalAddress(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080})
	if err != nil {
		t.Fatalf("error listening: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("error closing connection: %v", err)
		}
	}()

	go func() {
		for {
			_, _, err := conn.ReadFromUDP(make([]byte, 1024))
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				t.Errorf("error reading from udp: %v", err)
			}
		}
	}()

	for _, testCase := range testUdpDialWithOptionalLocalAddressCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := util.UdpDialWithOptionalLocalAddress(testCase.remoteAddress, testCase.remotePort, testCase.localAddress)
			if err != nil {
				t.Fatalf("error dialing: %v", err)
			}
		})
	}
}
