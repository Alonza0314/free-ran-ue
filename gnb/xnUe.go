package gnb

import (
	"net"
)

type XnUe struct {
	imsi string

	ulTeid []byte
	dlTeid []byte

	dataPlaneAddress *net.UDPAddr
}

func NewXnUe(imsi string, dlTeid []byte, dataPlaneAddress *net.UDPAddr) *XnUe {
	return &XnUe{
		imsi: imsi,

		ulTeid: []byte{},
		dlTeid: dlTeid,

		dataPlaneAddress: dataPlaneAddress,
	}
}

func (x *XnUe) Release(teidGenerator *TeidGenerator) {
	teidGenerator.ReleaseTeid(x.dlTeid)
}

func (x *XnUe) GetIMSI() string {
	return x.imsi
}

func (x *XnUe) GetUlTeid() []byte {
	return x.ulTeid
}

func (x *XnUe) GetDlTeid() []byte {
	return x.dlTeid
}

func (x *XnUe) GetDataPlaneAddress() *net.UDPAddr {
	return x.dataPlaneAddress
}

func (x *XnUe) SetUlTeid(ulTeid []byte) {
	x.ulTeid = ulTeid
}

func (x *XnUe) SetDataPlaneAddress(dataPlaneAddress *net.UDPAddr) {
	x.dataPlaneAddress = dataPlaneAddress
}
