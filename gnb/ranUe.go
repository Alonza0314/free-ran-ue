package gnb

import (
	"fmt"
	"net"
	"sync"

	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/free5gc/aper"
	"github.com/free5gc/nas/nasType"
)

type RanUeNgapIdGenerator struct {
	usedRanUeIds sync.Map
	mtx          sync.Mutex
}

func NewRanUeNgapIdGenerator() *RanUeNgapIdGenerator {
	return &RanUeNgapIdGenerator{
		usedRanUeIds: sync.Map{},
		mtx:          sync.Mutex{},
	}
}

func (g *RanUeNgapIdGenerator) AllocateRanUeId() int64 {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	for i := 1; i <= 65535; i++ {
		if _, exists := g.usedRanUeIds.Load(int64(i)); !exists {
			g.usedRanUeIds.Store(int64(i), true)
			return int64(i)
		}
	}

	return -1
}

func (g *RanUeNgapIdGenerator) ReleaseRanUeId(ranUeId int64) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	g.usedRanUeIds.Delete(ranUeId)
}

type RanUe struct {
	amfUeNgapId int64
	ranUeNgapId int64

	mobileIdentity5GS nasType.MobileIdentity5GS

	ulTeid aper.OctetString
	dlTeid aper.OctetString

	n1Conn           net.Conn
	dataPlaneAddress *net.UDPAddr

	pduSessionEstablishmentCompleteChan    chan struct{}
	ueContextReleaseCompleteChan           chan struct{}
	pduSessionModifyIndicationCompleteChan chan struct{}

	nrdcIndicator    bool
	nrdcIndicatorMtx sync.Mutex
}

func NewRanUe(n1Conn net.Conn, ranUeNgapIdGenerator *RanUeNgapIdGenerator) *RanUe {
	ranUeId := ranUeNgapIdGenerator.AllocateRanUeId()
	if ranUeId == -1 {
		panic("Failed to allocate ranUeId")
	}

	return &RanUe{
		amfUeNgapId: -1,
		ranUeNgapId: ranUeId,

		mobileIdentity5GS: nasType.MobileIdentity5GS{},

		n1Conn: n1Conn,

		pduSessionEstablishmentCompleteChan:    make(chan struct{}),
		ueContextReleaseCompleteChan:           make(chan struct{}),
		pduSessionModifyIndicationCompleteChan: make(chan struct{}),

		nrdcIndicator:    false,
		nrdcIndicatorMtx: sync.Mutex{},
	}
}

func (r *RanUe) Release(ranUeNgapIdGenerator *RanUeNgapIdGenerator, teidGenerator *TeidGenerator) {
	ranUeNgapIdGenerator.ReleaseRanUeId(r.ranUeNgapId)
	teidGenerator.ReleaseTeid(r.dlTeid)
	close(r.pduSessionEstablishmentCompleteChan)
	close(r.ueContextReleaseCompleteChan)
	close(r.pduSessionModifyIndicationCompleteChan)
}

func (r *RanUe) GetAmfUeId() int64 {
	return r.amfUeNgapId
}

func (r *RanUe) GetRanUeId() int64 {
	return r.ranUeNgapId
}

func (r *RanUe) GetMobileIdentityIMSI() string {
	suci := r.mobileIdentity5GS.GetSUCI()
	return fmt.Sprintf("%s%s%s%s", constant.UE_IMSI_PREFIX, suci[7:10], suci[11:13], suci[20:])
}

func (r *RanUe) GetUlTeid() aper.OctetString {
	return r.ulTeid
}

func (r *RanUe) GetDlTeid() aper.OctetString {
	return r.dlTeid
}

func (r *RanUe) GetN1Conn() net.Conn {
	return r.n1Conn
}

func (r *RanUe) GetDataPlaneAddress() *net.UDPAddr {
	return r.dataPlaneAddress
}

func (r *RanUe) SetAmfUeId(amfUeId int64) {
	r.amfUeNgapId = amfUeId
}

func (r *RanUe) SetRanUeId(ranUeId int64) {
	r.ranUeNgapId = ranUeId
}

func (r *RanUe) SetMobileIdentity5GS(mobileIdentity5GS nasType.MobileIdentity5GS) {
	r.mobileIdentity5GS = mobileIdentity5GS
}

func (r *RanUe) SetUlTeid(ulTeid aper.OctetString) {
	r.ulTeid = ulTeid
}

func (r *RanUe) SetDlTeid(dlTeid aper.OctetString) {
	r.dlTeid = dlTeid
}

func (r *RanUe) SetDataPlaneAddress(dataPlaneAddress *net.UDPAddr) {
	r.dataPlaneAddress = dataPlaneAddress
}

func (r *RanUe) GetPduSessionEstablishmentCompleteChan() chan struct{} {
	return r.pduSessionEstablishmentCompleteChan
}

func (r *RanUe) GetUeContextReleaseCompleteChan() chan struct{} {
	return r.ueContextReleaseCompleteChan
}

func (r *RanUe) GetPduSessionModifyIndicationCompleteChan() chan struct{} {
	return r.pduSessionModifyIndicationCompleteChan
}

func (r *RanUe) IsNrdcActivated() bool {
	r.nrdcIndicatorMtx.Lock()
	defer r.nrdcIndicatorMtx.Unlock()
	return r.nrdcIndicator
}

func (r *RanUe) ActivateNrdc() {
	r.nrdcIndicatorMtx.Lock()
	defer r.nrdcIndicatorMtx.Unlock()
	r.nrdcIndicator = true
}

func (r *RanUe) DeactivateNrdc() {
	r.nrdcIndicatorMtx.Lock()
	defer r.nrdcIndicatorMtx.Unlock()
	r.nrdcIndicator = false
}
