package gnb

import (
	"net"

	"github.com/free5gc/aper"
)

type XnUe struct {
	imsi string

	ulTeid aper.OctetString
	dlTeid aper.OctetString

	dataPlaneConn net.Conn
}

func NewXnUe(imsi string, dlTeid aper.OctetString, dataPlaneConn net.Conn) *XnUe {
	return &XnUe{
		imsi: imsi,

		ulTeid: aper.OctetString{},
		dlTeid: dlTeid,

		dataPlaneConn: dataPlaneConn,
	}
}

func (x *XnUe) Release(teidGenerator *TeidGenerator) {
	teidGenerator.ReleaseTeid(x.dlTeid)
}

func (x *XnUe) GetIMSI() string {
	return x.imsi
}

func (x *XnUe) GetUlTeid() aper.OctetString {
	return x.ulTeid
}

func (x *XnUe) GetDlTeid() aper.OctetString {
	return x.dlTeid
}

func (x *XnUe) GetDataPlaneConn() net.Conn {
	return x.dataPlaneConn
}

func (x *XnUe) SetUlTeid(ulTeid aper.OctetString) {
	x.ulTeid = ulTeid
}

func (x *XnUe) SetDataPlaneConn(dataPlaneConn net.Conn) {
	x.dataPlaneConn = dataPlaneConn
}
