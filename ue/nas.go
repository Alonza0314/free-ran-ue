package ue

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"

	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasConvert"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/openapi/models"
)

func nasDecode(ue *Ue, securityHeaderType uint8, payload []byte) (*nas.Message, error) {
	if payload == nil {
		return nil, errors.New("nas payload is nil")
	}

	msg := new(nas.Message)
	msg.SecurityHeaderType = uint8(nas.GetSecurityHeaderType(payload) & 0x0f)
	if securityHeaderType == nas.SecurityHeaderTypePlainNas {
		return msg, msg.PlainNasDecode(&payload)
	} else if ue.integrityAlgorithm == security.AlgIntegrity128NIA0 {
		payload = payload[3:]
		if err := security.NASEncrypt(ue.cipheringAlgorithm, ue.kNasEnc, ue.dlCount.Get(), ue.getBearerType(), security.DirectionDownlink, payload); err != nil {
			return nil, err
		}
		return msg, msg.PlainNasDecode(&payload)
	} else {
		securityHeader := payload[0:6]
		sequenceNumber := payload[6]
		receivedMac32 := securityHeader[2:]

		payload = payload[6:]

		ciphered := false
		switch msg.SecurityHeaderType {
		case nas.SecurityHeaderTypeIntegrityProtected:
		case nas.SecurityHeaderTypeIntegrityProtectedAndCiphered:
			ciphered = true
		case nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext:
			ue.dlCount.Set(0, 0)
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ciphered = true
			ue.dlCount.Set(0, 0)
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeader.SecurityHeaderType)
		}

		if ue.dlCount.SQN() > sequenceNumber {
			ue.dlCount.SetOverflow(ue.dlCount.Overflow() + 1)
		}
		ue.dlCount.SetSQN(sequenceNumber)

		if mac32, err := security.NASMacCalculate(ue.integrityAlgorithm, ue.kNasInt, ue.dlCount.Get(), ue.getBearerType(), security.DirectionDownlink, payload); err != nil {
			return nil, err
		} else {
			if !reflect.DeepEqual(mac32, receivedMac32) {
				return nil, fmt.Errorf("NAS MAC verification failed(0x%x != 0x%x)", mac32, receivedMac32)
			}
		}

		payload = payload[1:]
		if ciphered {
			if err := security.NASEncrypt(ue.cipheringAlgorithm, ue.kNasEnc, ue.dlCount.Get(), ue.getBearerType(),
				security.DirectionDownlink, payload); err != nil {
				return nil, err
			}
		}

		return msg, msg.PlainNasDecode(&payload)
	}
}

func nasEncode(nasMessage *nas.Message, securityContextAvailable bool, newSecurityContext bool, ue *Ue) ([]byte, error) {
	if nasMessage == nil {
		return nil, errors.New("nasMessage is nil")
	}

	if !securityContextAvailable {
		return nasMessage.PlainNasEncode()
	}

	if newSecurityContext {
		ue.ulCount.Set(0, 0)
		ue.dlCount.Set(0, 0)
	}

	sequenceNumber := ue.ulCount.SQN()
	payload, err := nasMessage.PlainNasEncode()
	if err != nil {
		return nil, err
	}
	if nasMessage.SecurityHeader.SecurityHeaderType != nas.SecurityHeaderTypeIntegrityProtected && nasMessage.SecurityHeader.SecurityHeaderType != nas.SecurityHeaderTypePlainNas {
		if err = security.NASEncrypt(ue.cipheringAlgorithm, ue.kNasEnc, ue.ulCount.Get(), ue.getBearerType(), security.DirectionUplink, payload); err != nil {
			return nil, err
		}
	}

	payload = append([]byte{sequenceNumber}, payload[:]...)

	mac32, err := security.NASMacCalculate(ue.integrityAlgorithm, ue.kNasInt, ue.ulCount.Get(), ue.getBearerType(), security.DirectionUplink, payload)
	if err != nil {
		return nil, err
	}
	payload = append(mac32, payload[:]...)

	msgSecurityHeader := []byte{nasMessage.SecurityHeader.ProtocolDiscriminator, nasMessage.SecurityHeader.SecurityHeaderType}
	payload = append(msgSecurityHeader, payload[:]...)

	return payload, nil
}

func buildUeMobileIdentity5GS(supi string) nasType.MobileIdentity5GS {
	supiBytes := util.SupiToBytes(supi)
	return nasType.MobileIdentity5GS{
		Len:    uint16(len(supiBytes)),
		Buffer: supiBytes,
	}
}

func buildUeSecurityCapability(cipheringAlgorithm uint8, integrityAlgorithm uint8) nasType.UESecurityCapability {
	ueSecurityCapability := nasType.UESecurityCapability{
		Iei:    nasMessage.RegistrationRequestUESecurityCapabilityType,
		Len:    2,
		Buffer: []byte{0x00, 0x00},
	}

	switch cipheringAlgorithm {
	case security.AlgCiphering128NEA0:
		ueSecurityCapability.SetEA0_5G(1)
	case security.AlgCiphering128NEA1:
		ueSecurityCapability.SetEA1_128_5G(1)
	case security.AlgCiphering128NEA2:
		ueSecurityCapability.SetEA2_128_5G(1)
	case security.AlgCiphering128NEA3:
		ueSecurityCapability.SetEA3_128_5G(1)
	}

	switch integrityAlgorithm {
	case security.AlgIntegrity128NIA0:
		ueSecurityCapability.SetIA0_5G(1)
	case security.AlgIntegrity128NIA1:
		ueSecurityCapability.SetIA1_128_5G(1)
	case security.AlgIntegrity128NIA2:
		ueSecurityCapability.SetIA2_128_5G(1)
	case security.AlgIntegrity128NIA3:
		ueSecurityCapability.SetIA3_128_5G(1)
	}

	return ueSecurityCapability
}

func buildUeRegistrationRequest(registrationType uint8, mobileIdentity5GS *nasType.MobileIdentity5GS, requestedNSSAI *nasType.RequestedNSSAI, ueSecurityCapability *nasType.UESecurityCapability, capability5GMM *nasType.Capability5GMM, nasMessageContainer []uint8, uplinkDataStatus *nasType.UplinkDataStatus) ([]byte, error) {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeRegistrationRequest)

	registrationRequest := nasMessage.NewRegistrationRequest(0)
	registrationRequest.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	registrationRequest.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	registrationRequest.SpareHalfOctetAndSecurityHeaderType.SetSpareHalfOctet(0x00)
	registrationRequest.RegistrationRequestMessageIdentity.SetMessageType(nas.MsgTypeRegistrationRequest)
	registrationRequest.NgksiAndRegistrationType5GS.SetTSC(nasMessage.TypeOfSecurityContextFlagNative)
	registrationRequest.NgksiAndRegistrationType5GS.SetNasKeySetIdentifiler(0x7)
	registrationRequest.NgksiAndRegistrationType5GS.SetFOR(1)
	registrationRequest.NgksiAndRegistrationType5GS.SetRegistrationType5GS(registrationType)
	registrationRequest.MobileIdentity5GS = *mobileIdentity5GS

	registrationRequest.UESecurityCapability = ueSecurityCapability
	registrationRequest.Capability5GMM = capability5GMM
	registrationRequest.RequestedNSSAI = requestedNSSAI
	registrationRequest.UplinkDataStatus = uplinkDataStatus

	if nasMessageContainer != nil {
		registrationRequest.NASMessageContainer = nasType.NewNASMessageContainer(
			nasMessage.RegistrationRequestNASMessageContainerType)
		registrationRequest.NASMessageContainer.SetLen(uint16(len(nasMessageContainer)))
		registrationRequest.NASMessageContainer.SetNASMessageContainerContents(nasMessageContainer)
	}

	m.GmmMessage.RegistrationRequest = registrationRequest

	request := new(bytes.Buffer)
	if err := m.GmmMessageEncode(request); err != nil {
		return nil, err
	}

	return request.Bytes(), nil
}

func getUeRegistrationRequest(registrationType uint8, mobileIdentity5GS *nasType.MobileIdentity5GS, requestedNSSAI *nasType.RequestedNSSAI, ueSecurityCapability *nasType.UESecurityCapability, capability5GMM *nasType.Capability5GMM, nasMessageContainer []uint8, uplinkDataStatus *nasType.UplinkDataStatus) ([]byte, error) {
	return buildUeRegistrationRequest(registrationType, mobileIdentity5GS, requestedNSSAI, ueSecurityCapability, capability5GMM, nasMessageContainer, uplinkDataStatus)
}

func buildAuthenticationResponse(authenticationResponseParam []byte) ([]byte, error) {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeAuthenticationResponse)

	authenticationResponse := nasMessage.NewAuthenticationResponse(0)
	authenticationResponse.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(
		nasMessage.Epd5GSMobilityManagementMessage)
	authenticationResponse.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	authenticationResponse.SpareHalfOctetAndSecurityHeaderType.SetSpareHalfOctet(0)
	authenticationResponse.AuthenticationResponseMessageIdentity.SetMessageType(nas.MsgTypeAuthenticationResponse)

	if len(authenticationResponseParam) > 0 {
		authenticationResponse.AuthenticationResponseParameter = nasType.NewAuthenticationResponseParameter(
			nasMessage.AuthenticationResponseAuthenticationResponseParameterType)
		authenticationResponse.AuthenticationResponseParameter.SetLen(uint8(len(authenticationResponseParam)))
		copy(authenticationResponse.AuthenticationResponseParameter.Octet[:], authenticationResponseParam[0:16])
	}

	m.GmmMessage.AuthenticationResponse = authenticationResponse

	response := new(bytes.Buffer)
	if err := m.GmmMessageEncode(response); err != nil {
		return nil, err
	}

	return response.Bytes(), nil
}

func getAuthenticationResponse(authenticationResponseParam []byte) ([]byte, error) {
	return buildAuthenticationResponse(authenticationResponseParam)
}

func buildNasSecurityModeCompleteMessage(nasMessageContainer []byte) ([]byte, error) {
	m := nas.NewMessage()

	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeSecurityModeComplete)

	securityModeComplete := nasMessage.NewSecurityModeComplete(0)
	securityModeComplete.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)

	securityModeComplete.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	securityModeComplete.SpareHalfOctetAndSecurityHeaderType.SetSpareHalfOctet(0)
	securityModeComplete.SecurityModeCompleteMessageIdentity.SetMessageType(nas.MsgTypeSecurityModeComplete)

	securityModeComplete.IMEISV = nasType.NewIMEISV(nasMessage.SecurityModeCompleteIMEISVType)
	securityModeComplete.IMEISV.SetLen(9)
	securityModeComplete.SetOddEvenIdic(0)
	securityModeComplete.SetTypeOfIdentity(nasMessage.MobileIdentity5GSTypeImeisv)
	securityModeComplete.SetIdentityDigit1(1)
	securityModeComplete.SetIdentityDigitP_1(1)
	securityModeComplete.SetIdentityDigitP(1)

	if nasMessageContainer != nil {
		securityModeComplete.NASMessageContainer = nasType.NewNASMessageContainer(nasMessage.SecurityModeCompleteNASMessageContainerType)
		securityModeComplete.NASMessageContainer.SetLen(uint16(len(nasMessageContainer)))
		securityModeComplete.NASMessageContainer.SetNASMessageContainerContents(nasMessageContainer)
	}

	m.GmmMessage.SecurityModeComplete = securityModeComplete

	completeMessage := new(bytes.Buffer)
	if err := m.GmmMessageEncode(completeMessage); err != nil {
		return nil, err
	}

	return completeMessage.Bytes(), nil
}

func getNasSecurityModeCompleteMessage(nasMessageContainer []byte) ([]byte, error) {
	return buildNasSecurityModeCompleteMessage(nasMessageContainer)
}

func buildNasRegistrationCompleteMessage(sorTransparentContainer []byte) ([]byte, error) {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeRegistrationComplete)

	registrationComplete := nasMessage.NewRegistrationComplete(0)
	registrationComplete.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(
		nasMessage.Epd5GSMobilityManagementMessage)
	registrationComplete.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	registrationComplete.SpareHalfOctetAndSecurityHeaderType.SetSpareHalfOctet(0)
	registrationComplete.RegistrationCompleteMessageIdentity.SetMessageType(nas.MsgTypeRegistrationComplete)

	if sorTransparentContainer != nil {
		registrationComplete.SORTransparentContainer = nasType.NewSORTransparentContainer(
			nasMessage.RegistrationCompleteSORTransparentContainerType)
		registrationComplete.SORTransparentContainer.SetLen(uint16(len(sorTransparentContainer)))
		registrationComplete.SORTransparentContainer.SetSORContent(sorTransparentContainer)
	}

	m.GmmMessage.RegistrationComplete = registrationComplete

	completeMessage := new(bytes.Buffer)
	if err := m.GmmMessageEncode(completeMessage); err != nil {
		return nil, err
	}

	return completeMessage.Bytes(), nil
}

func getNasRegistrationCompleteMessage(nasMessageContainer []byte) ([]byte, error) {
	return buildNasRegistrationCompleteMessage(nasMessageContainer)
}

func buildPduSessionEstablishmentRequest(pduSessionId uint8) ([]byte, error) {
	m := nas.NewMessage()
	m.GsmMessage = nas.NewGsmMessage()
	m.GsmHeader.SetMessageType(nas.MsgTypePDUSessionEstablishmentRequest)

	pduSessionEstablishmentRequest := nasMessage.NewPDUSessionEstablishmentRequest(0)
	pduSessionEstablishmentRequest.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSSessionManagementMessage)
	pduSessionEstablishmentRequest.SetMessageType(nas.MsgTypePDUSessionEstablishmentRequest)
	pduSessionEstablishmentRequest.PDUSessionID.SetPDUSessionID(pduSessionId)
	pduSessionEstablishmentRequest.PTI.SetPTI(0x00)
	pduSessionEstablishmentRequest.IntegrityProtectionMaximumDataRate.SetMaximumDataRatePerUEForUserPlaneIntegrityProtectionForDownLink(0xff)
	pduSessionEstablishmentRequest.IntegrityProtectionMaximumDataRate.SetMaximumDataRatePerUEForUserPlaneIntegrityProtectionForUpLink(0xff)

	pduSessionEstablishmentRequest.PDUSessionType = nasType.NewPDUSessionType(nasMessage.PDUSessionEstablishmentRequestPDUSessionTypeType)
	pduSessionEstablishmentRequest.PDUSessionType.SetPDUSessionTypeValue(uint8(0x01)) //IPv4 type

	pduSessionEstablishmentRequest.SSCMode = nasType.NewSSCMode(nasMessage.PDUSessionEstablishmentRequestSSCModeType)
	pduSessionEstablishmentRequest.SSCMode.SetSSCMode(uint8(0x01)) //SSC Mode 1

	pduSessionEstablishmentRequest.ExtendedProtocolConfigurationOptions = nasType.NewExtendedProtocolConfigurationOptions(nasMessage.PDUSessionEstablishmentRequestExtendedProtocolConfigurationOptionsType)
	protocolConfigurationOptions := nasConvert.NewProtocolConfigurationOptions()
	protocolConfigurationOptions.AddIPAddressAllocationViaNASSignallingUL()
	protocolConfigurationOptions.AddDNSServerIPv4AddressRequest()
	protocolConfigurationOptions.AddDNSServerIPv6AddressRequest()
	pcoContents := protocolConfigurationOptions.Marshal()
	pcoContentsLength := len(pcoContents)
	pduSessionEstablishmentRequest.ExtendedProtocolConfigurationOptions.SetLen(uint16(pcoContentsLength))
	pduSessionEstablishmentRequest.ExtendedProtocolConfigurationOptions.SetExtendedProtocolConfigurationOptionsContents(pcoContents)

	m.GsmMessage.PDUSessionEstablishmentRequest = pduSessionEstablishmentRequest

	request := new(bytes.Buffer)
	if err := m.GsmMessageEncode(request); err != nil {
		return nil, err
	}

	return request.Bytes(), nil
}

func getPduSessionEstablishmentRequest(pduSessionId uint8) ([]byte, error) {
	return buildPduSessionEstablishmentRequest(pduSessionId)
}

func buildUlNasTransportMessage(nasMessageContainer []byte, pduSessionId uint8, requestType uint8, dnn string, sNssai *models.Snssai) ([]byte, error) {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeULNASTransport)

	ulNasTransport := nasMessage.NewULNASTransport(0)
	ulNasTransport.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	ulNasTransport.SetMessageType(nas.MsgTypeULNASTransport)
	ulNasTransport.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	ulNasTransport.PduSessionID2Value = new(nasType.PduSessionID2Value)
	ulNasTransport.PduSessionID2Value.SetIei(nasMessage.ULNASTransportPduSessionID2ValueType)
	ulNasTransport.PduSessionID2Value.SetPduSessionID2Value(pduSessionId)
	ulNasTransport.RequestType = new(nasType.RequestType)
	ulNasTransport.RequestType.SetIei(nasMessage.ULNASTransportRequestTypeType)
	ulNasTransport.RequestType.SetRequestTypeValue(requestType)

	if dnn != "" {
		ulNasTransport.DNN = new(nasType.DNN)
		ulNasTransport.DNN.SetIei(nasMessage.ULNASTransportDNNType)
		ulNasTransport.DNN.SetDNN(dnn)
	}

	if sNssai != nil {
		var sdTemp [3]uint8
		sd, err := hex.DecodeString(sNssai.Sd)
		if err != nil {
			return nil, fmt.Errorf("sNssai decode error: %v", err)
		}

		copy(sdTemp[:], sd)
		ulNasTransport.SNSSAI = nasType.NewSNSSAI(nasMessage.ULNASTransportSNSSAIType)
		ulNasTransport.SNSSAI.SetLen(4)
		ulNasTransport.SNSSAI.SetSST(uint8(sNssai.Sst))
		ulNasTransport.SNSSAI.SetSD(sdTemp)
	}

	ulNasTransport.SpareHalfOctetAndPayloadContainerType.SetPayloadContainerType(nasMessage.PayloadContainerTypeN1SMInfo)
	ulNasTransport.PayloadContainer.SetLen(uint16(len(nasMessageContainer)))
	ulNasTransport.PayloadContainer.SetPayloadContainerContents(nasMessageContainer)

	m.GmmMessage.ULNASTransport = ulNasTransport

	message := new(bytes.Buffer)
	if err := m.GmmMessageEncode(message); err != nil {
		return nil, err
	}

	return message.Bytes(), nil
}

func getUlNasTransportMessage(nasMessageContainer []byte, pduSessionId uint8, requestType uint8, dnn string, sNssai *models.Snssai) ([]byte, error) {
	return buildUlNasTransportMessage(nasMessageContainer, pduSessionId, requestType, dnn, sNssai)
}

func buildUeDeRegistrationRequest(accessType uint8, switchOff uint8, ngKsi uint8, mobileIdentity5GS nasType.MobileIdentity5GS) ([]byte, error) {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeDeregistrationRequestUEOriginatingDeregistration)

	deregistrationRequest := nasMessage.NewDeregistrationRequestUEOriginatingDeregistration(0)
	deregistrationRequest.ExtendedProtocolDiscriminator.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	deregistrationRequest.SpareHalfOctetAndSecurityHeaderType.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	deregistrationRequest.SpareHalfOctetAndSecurityHeaderType.SetSpareHalfOctet(0)
	deregistrationRequest.DeregistrationRequestMessageIdentity.SetMessageType(nas.MsgTypeDeregistrationRequestUEOriginatingDeregistration)

	deregistrationRequest.NgksiAndDeregistrationType.SetAccessType(accessType)
	deregistrationRequest.NgksiAndDeregistrationType.SetSwitchOff(switchOff)
	deregistrationRequest.NgksiAndDeregistrationType.SetReRegistrationRequired(0)
	deregistrationRequest.NgksiAndDeregistrationType.SetTSC(ngKsi)
	deregistrationRequest.NgksiAndDeregistrationType.SetNasKeySetIdentifiler(ngKsi)
	deregistrationRequest.MobileIdentity5GS.SetLen(mobileIdentity5GS.GetLen())
	deregistrationRequest.MobileIdentity5GS.SetMobileIdentity5GSContents(mobileIdentity5GS.GetMobileIdentity5GSContents())

	m.GmmMessage.DeregistrationRequestUEOriginatingDeregistration = deregistrationRequest

	request := new(bytes.Buffer)
	if err := m.GmmMessageEncode(request); err != nil {
		return nil, err
	}

	return request.Bytes(), nil
}

func getUeDeRegistrationRequest(accessType uint8, switchOff uint8, ngKsi uint8, mobileIdentity5GS nasType.MobileIdentity5GS) ([]byte, error) {
	return buildUeDeRegistrationRequest(accessType, switchOff, ngKsi, mobileIdentity5GS)
}

func getNasPduFromNasPduSessionEstablishmentAccept(nasPduSessionEstablishmentAccept *nas.Message) (*nas.Message, error) {
	content := nasPduSessionEstablishmentAccept.DLNASTransport.GetPayloadContainerContents()

	nasMessage := new(nas.Message)
	if err := nasMessage.PlainNasDecode(&content); err != nil {
		return nil, err
	}

	return nasMessage, nil
}
