package ue

import (
	"errors"
	"fmt"

	"github.com/free-ran-ue/util"
	"github.com/free5gc/nas/ie"
	"github.com/free5gc/nas/message"
	"github.com/free5gc/openapi/models"
)

func nasDecode(ue *Ue, payload []byte) (message.Message, error) {
	if payload == nil {
		return nil, errors.New("nas payload is nil")
	}

	return message.Parse(payload, ue.secCtx)
}

func nasEncode(m message.Message, sc *message.SecCtx, hdrType message.SecHdrType) ([]byte, error) {
	if m == nil {
		return nil, errors.New("nasMessage is nil")
	}

	return message.Marshal(m, sc, hdrType)
}

func buildUeMobileIdentity5GS(nccLength, mncLength int, supi string) (*ie.MobileId5GS, error) {
	supiBytes := util.SupiToBytes(nccLength, mncLength, supi)

	mobileIdentity5GS := new(ie.MobileId5GS)
	if err := mobileIdentity5GS.UnmarshalBinary(supiBytes); err != nil {
		return nil, fmt.Errorf("error unmarshal mobile identity 5gs: %+v", err)
	}
	return mobileIdentity5GS, nil
}

func buildUeSecurityCapability(cipheringAlgorithm ie.AlgCiphering, integrityAlgorithm ie.AlgIntegrity) *ie.UESecCapability {
	ueSecurityCapability := &ie.UESecCapability{
		Length: 2,
	}

	switch cipheringAlgorithm {
	case message.AlgCiphering128NEA0:
		ueSecurityCapability.EA05G = true
	case message.AlgCiphering128NEA1:
		ueSecurityCapability.EA1_128_5G = true
	case message.AlgCiphering128NEA2:
		ueSecurityCapability.EA2_128_5G = true
	case message.AlgCiphering128NEA3:
		ueSecurityCapability.EA3_128_5G = true
	}

	switch integrityAlgorithm {
	case message.AlgIntegrity128NIA0:
		ueSecurityCapability.IA05G = true
	case message.AlgIntegrity128NIA1:
		ueSecurityCapability.IA1_128_5G = true
	case message.AlgIntegrity128NIA2:
		ueSecurityCapability.IA2_128_5G = true
	case message.AlgIntegrity128NIA3:
		ueSecurityCapability.IA3_128_5G = true
	}

	return ueSecurityCapability
}

// buildUeRegistrationRequest returns the plain-encoded bytes of a RegReq. The
// caller always either sends it unprotected (initial registration, before a
// security context exists) or embeds it as a NAS message container, so no
// security wrapping is ever applied to this message directly.
func buildUeRegistrationRequest(registrationType uint8, mobileIdentity5GS *ie.MobileId5GS, requestedNSSAI *ie.NSSAI, ueSecurityCapability *ie.UESecCapability, capability5GMM *ie.Capability5GMM, nasMessageContainer []uint8, uplinkDataStatus *ie.UplinkDataStatus) ([]byte, error) {
	registrationRequest := &message.RegReq{
		RegType5GS: &ie.RegType5GS{
			FOR_Pending: true,
			Value:       registrationType,
		},
		Ngksi: &ie.NASKeySetId{
			Tsc: ie.SecCtxTypeNative,
			Ksi: ie.NASKeyNA,
		},
		MobileId5GS:      mobileIdentity5GS,
		UESecCapability:  ueSecurityCapability,
		Capability5GMM:   capability5GMM,
		ReqNSSAI:         requestedNSSAI,
		UplinkDataStatus: uplinkDataStatus,
	}

	if nasMessageContainer != nil {
		registrationRequest.NASMsgCntr = &ie.NASMsgCntr{Contents: nasMessageContainer}
	}

	return registrationRequest.MarshalBinary()
}

func getUeRegistrationRequest(registrationType uint8, mobileIdentity5GS *ie.MobileId5GS, requestedNSSAI *ie.NSSAI, ueSecurityCapability *ie.UESecCapability, capability5GMM *ie.Capability5GMM, nasMessageContainer []uint8, uplinkDataStatus *ie.UplinkDataStatus) ([]byte, error) {
	return buildUeRegistrationRequest(registrationType, mobileIdentity5GS, requestedNSSAI, ueSecurityCapability, capability5GMM, nasMessageContainer, uplinkDataStatus)
}

// buildAuthenticationResponse returns the plain-encoded bytes of an AuthRsp.
// It is always sent unprotected, per 3GPP TS 24.501, since the security
// context is not yet active at this point of the registration procedure.
func buildAuthenticationResponse(authenticationResponseParam []byte) ([]byte, error) {
	authenticationResponse := &message.AuthRsp{}

	if len(authenticationResponseParam) > 0 {
		authenticationResponse.AuthRspParam = &ie.AuthRspParam{
			Res: authenticationResponseParam,
		}
	}

	return authenticationResponse.MarshalBinary()
}

func getAuthenticationResponse(authenticationResponseParam []byte) ([]byte, error) {
	return buildAuthenticationResponse(authenticationResponseParam)
}

func buildNasSecurityModeCompleteMessage(nasMessageContainer []byte) *message.SecModeComplete {
	imeisv := &ie.MobileId5GS{
		TypeOfId:     ie.IdType_5GS_IMEISV,
		OddEvenIndic: ie.EvenNumOfIdDigit,
	}
	imeisv.IMEISV[0] = 1
	imeisv.IMEISV[14] = 1
	imeisv.IMEISV[15] = 1

	securityModeComplete := &message.SecModeComplete{
		IMEISV: imeisv,
	}

	if nasMessageContainer != nil {
		securityModeComplete.NASMsgCntr = &ie.NASMsgCntr{Contents: nasMessageContainer}
	}

	return securityModeComplete
}

func getNasSecurityModeCompleteMessage(nasMessageContainer []byte) *message.SecModeComplete {
	return buildNasSecurityModeCompleteMessage(nasMessageContainer)
}

func buildNasRegistrationCompleteMessage() *message.RegComplete {
	return &message.RegComplete{}
}

func getNasRegistrationCompleteMessage() *message.RegComplete {
	return buildNasRegistrationCompleteMessage()
}

// buildPduSessionEstablishmentRequest returns the plain-encoded bytes of a
// PDUSessEstReq. It is never sent standalone; it is always embedded as the
// payload container of an UL NAS Transport message.
func buildPduSessionEstablishmentRequest(pduSessionId uint8) ([]byte, error) {
	pduSessionEstablishmentRequest := &message.PDUSessEstReq{
		PDUSessId: pduSessionId,
		PTI:       0x00,
		IntegrityProtectionMaxDataRate: &ie.IntegrityProtectionMaxDataRate{
			Uplink:   0xff,
			Downlink: 0xff,
		},
		PDUSessType: &ie.PDUSessType{
			Value: ie.PDUSessType_IPv4,
		},
		SSCMode: &ie.SSCMode{
			Mode: ie.SSCMODE1,
		},
		ExtendedProtCfgOpts: &ie.ExtendedProtCfgOpts{
			FromMs: &ie.ExtCfgOptFromMs{
				DNSV4Req: true,
				DNSV6Req: true,
			},
		},
	}

	return pduSessionEstablishmentRequest.MarshalBinary()
}

func getPduSessionEstablishmentRequest(pduSessionId uint8) ([]byte, error) {
	return buildPduSessionEstablishmentRequest(pduSessionId)
}

func buildUlNasTransportMessage(nasMessageContainer []byte, pduSessionId uint8, requestType ie.ConstReqType, dnn string, sNssai *models.Snssai) *message.ULNASTransport {
	ulNasTransport := &message.ULNASTransport{
		PayloadCntrType: &ie.PayloadCntrType{
			Value: ie.PayloadCntrType_N1SMInfo,
		},
		PayloadCntr: &ie.PayloadCntr{
			Pct:      ie.PayloadCntrType_N1SMInfo,
			Contents: nasMessageContainer,
		},
		PDUSessID: &ie.PDUSessId2{
			Value: pduSessionId,
		},
		ReqType: &ie.ReqType{
			Value: requestType,
		},
	}

	if dnn != "" {
		ulNasTransport.DNN = &ie.DNN{Value: dnn}
	}

	if sNssai != nil {
		ulNasTransport.SNSSAI = &ie.SNSSAI{
			SST: uint8(sNssai.Sst),
			SD:  sNssai.Sd,
		}
	}

	return ulNasTransport
}

func getUlNasTransportMessage(nasMessageContainer []byte, pduSessionId uint8, requestType ie.ConstReqType, dnn string, sNssai *models.Snssai) *message.ULNASTransport {
	return buildUlNasTransportMessage(nasMessageContainer, pduSessionId, requestType, dnn, sNssai)
}

func buildUeDeRegistrationRequest(accessType uint8, switchOff uint8, ngKsi uint8, mobileIdentity5GS *ie.MobileId5GS) *message.DeregReqUEOrig {
	return &message.DeregReqUEOrig{
		DeregType: &ie.DeregType{
			Switchoff:     switchOff&0x1 == 1,
			ReregRequired: false,
			AccessType:    accessType & 0x3,
		},
		Ngksi: &ie.NASKeySetId{
			Tsc: ie.SecCtxType(ngKsi & 0x1),
			Ksi: ngKsi & 0x7,
		},
		MobileId5GS: mobileIdentity5GS,
	}
}

func getUeDeRegistrationRequest(accessType uint8, switchOff uint8, ngKsi uint8, mobileIdentity5GS *ie.MobileId5GS) *message.DeregReqUEOrig {
	return buildUeDeRegistrationRequest(accessType, switchOff, ngKsi, mobileIdentity5GS)
}

func getNasPduFromNasPduSessionEstablishmentAccept(dlNasTransport *message.DLNASTransport) (message.GSMMessage, error) {
	if dlNasTransport.PayloadCntr == nil {
		return nil, errors.New("dl nas transport has no payload container")
	}

	return message.ParseGSM(dlNasTransport.PayloadCntr.Contents)
}
