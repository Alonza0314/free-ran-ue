package util_test

import (
	"errors"
	"net"
	"testing"

	"github.com/free-ran-ue/free-ran-ue/util"
)

var testTcpDialWithOptionalLocalAddressCases = []struct {
	name          string
	remoteAddress string
	remotePort    int
	localAddress  string
}{
	{
		name:          "testTcpDialWithoutLocalAddress",
		remoteAddress: "127.0.0.1",
		remotePort:    8080,
		localAddress:  "",
	},
	{
		name:          "testTcpDialWithLocalAddress",
		remoteAddress: "127.0.0.1",
		remotePort:    8080,
		localAddress:  "127.0.0.1",
	},
}

func TestTcpDialWithOptionalLocalAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("error listening: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Fatalf("error closing listener: %v", err)
		}
	}()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			t.Errorf("error accepting: %v", err)
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Errorf("error closing connection: %v", err)
			}
		}()
	}()

	for _, testCase := range testTcpDialWithOptionalLocalAddressCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := util.TcpDialWithOptionalLocalAddress(testCase.remoteAddress, testCase.remotePort, testCase.localAddress)
			if err != nil {
				t.Fatalf("error dialing: %v", err)
			}
		})
	}
}
