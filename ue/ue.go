package ue

import (
	"errors"
	"fmt"
	"net"

	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
)

type authentication struct {
	supi string

	cipheringAlgorithm uint8
	integrityAlgorithm uint8

	kNasEnc [16]byte
	kNasInt [16]byte
	kAmf    []uint8

	ulCount security.Count
	dlCount security.Count
}

type authenticationSubscription struct {
	authenticationMethod          models.AuthMethod
	encPermanentKey               string
	encOpcKey                     string
	authenticationManagementField string
	sequenceNumber                string
}

type Ue struct {
	ranIp   string
	ranPort int
	ranConn net.Conn

	mcc  string
	mnc  string
	msin string

	authentication

	accessType models.AccessType
	authenticationSubscription

	logger *logger.Logger
}

func NewUe(config *model.UeConfig, logger *logger.Logger) *Ue {
	supi := config.Ue.PlmnId.Mcc + config.Ue.PlmnId.Mnc + config.Ue.Msin

	var integrityAlgorithm uint8
	if config.Ue.IntegrityAlgorithm.Nia0 {
		integrityAlgorithm = security.AlgIntegrity128NIA0
	} else if config.Ue.IntegrityAlgorithm.Nia1 {
		integrityAlgorithm = security.AlgIntegrity128NIA1
	} else if config.Ue.IntegrityAlgorithm.Nia2 {
		integrityAlgorithm = security.AlgIntegrity128NIA2
	} else if config.Ue.IntegrityAlgorithm.Nia3 {
		integrityAlgorithm = security.AlgIntegrity128NIA3
	}

	var cipheringAlgorithm uint8
	if config.Ue.CipheringAlgorithm.Nea0 {
		cipheringAlgorithm = security.AlgCiphering128NEA0
	} else if config.Ue.CipheringAlgorithm.Nea1 {
		cipheringAlgorithm = security.AlgCiphering128NEA1
	} else if config.Ue.CipheringAlgorithm.Nea2 {
		cipheringAlgorithm = security.AlgCiphering128NEA2
	} else if config.Ue.CipheringAlgorithm.Nea3 {
		cipheringAlgorithm = security.AlgCiphering128NEA3
	}

	return &Ue{
		ranIp:   config.Ue.RanIp,
		ranPort: config.Ue.RanPort,

		mcc:  config.Ue.PlmnId.Mcc,
		mnc:  config.Ue.PlmnId.Mnc,
		msin: config.Ue.Msin,

		authentication: authentication{
			supi: supi,

			cipheringAlgorithm: cipheringAlgorithm,
			integrityAlgorithm: integrityAlgorithm,

			ulCount: security.Count{},
			dlCount: security.Count{},
		},

		accessType: models.AccessType(config.Ue.AccessType),
		authenticationSubscription: authenticationSubscription{
			authenticationMethod:          models.AuthMethod__5_G_AKA,
			encPermanentKey:               config.Ue.AuthenticationSubscription.EncPermanentKey,
			encOpcKey:                     config.Ue.AuthenticationSubscription.EncOpcKey,
			authenticationManagementField: config.Ue.AuthenticationSubscription.AuthenticationManagementField,
			sequenceNumber:                config.Ue.AuthenticationSubscription.SequenceNumber,
		},

		logger: logger,
	}
}

func (u *Ue) Start() error {
	u.logger.Info("UE", fmt.Sprintf("Starting UE: imsi-%s", u.supi))

	if err := u.connectToRan(); err != nil {
		u.logger.Error("UE", fmt.Sprintf("Error connecting to RAN: %v", err))
		return err
	}

	if err := u.processUeRegistration(); err != nil {
		u.logger.Error("UE", fmt.Sprintf("Error sending UE Registration Request: %v", err))
		return err
	}

	u.logger.Info("UE", "UE started")
	return nil
}

func (u *Ue) Stop() {
	u.logger.Info("UE", fmt.Sprintf("Stopping UE: imsi-%s", u.supi))

	if err := u.ranConn.Close(); err != nil {
		u.logger.Error("UE", fmt.Sprintf("Error closing RAN connection: %v", err))
	}
	u.logger.Info("UE", "UE stopped")
}

func (u *Ue) connectToRan() error {
	u.logger.Info("UE", "Connecting to RAN")

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", u.ranIp, u.ranPort))
	if err != nil {
		return err
	}
	u.ranConn = conn

	u.logger.Info("UE", fmt.Sprintf("Connected to RAN: %s:%d", u.ranIp, u.ranPort))
	return nil
}

func (u *Ue) processUeRegistration() error {
	u.logger.Info("NAS", fmt.Sprintf("Processing UE Registration"))

	mobileIdentity5GS := buildUeMobileIdentity5GS(u.supi)

	ueSecurityCapability := buildUeSecurityCapability(u.cipheringAlgorithm, u.integrityAlgorithm)

	// send ue registration request
	registrationRequest, err := getUeRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration, &mobileIdentity5GS, nil, &ueSecurityCapability, nil, nil, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get ue registration request: %+v", err))
	}
	u.logger.Debug("NAS", fmt.Sprintf("Get UE %s registration request", u.supi))

	if _, err := u.ranConn.Write(registrationRequest); err != nil {
		return errors.New(fmt.Sprintf("Error send ue registration request: %+v", err))
	}
	u.logger.Debug("NAS", fmt.Sprintf("Send UE %s registration request", u.supi))

	// receive nas authentication request
	nasAuthenticationRequestRaw := make([]byte, 1024)
	n, err := u.ranConn.Read(nasAuthenticationRequestRaw)
	if err != nil {
		return errors.New(fmt.Sprintf("Error read nas authentication request: %+v", err))
	}
	nasAuthenticationRequest, err := ngap.Decoder(nasAuthenticationRequestRaw[:n])
	if err != nil {
		return errors.New(fmt.Sprintf("Error decode nas authentication request: %+v", err))
	}
	if nasAuthenticationRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || nasAuthenticationRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return errors.New(fmt.Sprintf("Error NGAP nas authentication request: %+v", nasAuthenticationRequest))
	}

	nasPdu, err := getNasPdu(u, nasAuthenticationRequest.InitiatingMessage.Value.DownlinkNASTransport)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get nas pdu: %+v", err))
	} else {
		if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeAuthenticationRequest {
			return errors.New(fmt.Sprintf("Error nas pdu message type: %+v, expected authenticatoin request", nasPdu))
		}
	}
	u.logger.Debug("NAS", fmt.Sprintf("Receive NAS Authentication Request from RAN"))

	// calculate for RES* and send nas authentication response
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	kAmf, kenc, kint, resStar, err := deriveResStarAndSetKey(fmt.Sprintf("supi-%s", u.supi), u.cipheringAlgorithm, u.integrityAlgorithm, u.authenticationSubscription.sequenceNumber, u.authenticationSubscription.authenticationManagementField, u.authenticationSubscription.encPermanentKey, u.authenticationSubscription.encOpcKey, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")
	if err != nil {
		return errors.New(fmt.Sprintf("Error derive res star and set key: %+v", err))
	} else {
		u.kAmf = kAmf
		copy(u.kNasEnc[:], kenc[16:32])
		copy(u.kNasInt[:], kint[16:32])
	}

	authenticationResponse, err := getAuthenticationResponse(resStar)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get authentication response: %+v", err))
	}

	if _, err := u.ranConn.Write(authenticationResponse); err != nil {
		return errors.New(fmt.Sprintf("Error send authentication response: %+v", err))
	}
	u.logger.Debug("NAS", fmt.Sprintf("Send Authentication Response to RAN"))

	// receive nas security mode command message
	nasSecurityCommandRaw := make([]byte, 1024)
	n, err = u.ranConn.Read(nasSecurityCommandRaw)
	if err != nil {
		return errors.New(fmt.Sprintf("Error read nas security command: %+v", err))
	}
	nasSecurityCommand, err := ngap.Decoder(nasSecurityCommandRaw[:n])
	if err != nil {
		return errors.New(fmt.Sprintf("Error decode nas security command: %+v", err))
	}

	nasPdu, err = getNasPdu(u, nasSecurityCommand.InitiatingMessage.Value.DownlinkNASTransport)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get nas pdu: %+v", err))
	} else {
		if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeSecurityModeCommand {
			return errors.New(fmt.Sprintf("Error nas pdu message type: %+v, expected security mode command", nasPdu))
		}
	}
	u.logger.Debug("NAS", fmt.Sprintf("Receive NAS Security Mode Command from RAN"))

	// send nas security mode complete message
	registrationRequestWith5Gmm, err := getUeRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration, &mobileIdentity5GS, nil, &ueSecurityCapability, u.get5GmmCapability(), nil, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get ue registration request with 5GMM: %+v", err))
	}

	nasSecurityModeCompleteMessage, err := getNasSecurityModeCompleteMessage(registrationRequestWith5Gmm)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get nas security mode complete message: %+v", err))
	}

	encodedNasSecurityModeCompleteMessage, err := encodeNasPduWithSecurity(nasSecurityModeCompleteMessage, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, u, true, true)
	if err != nil {
		return errors.New(fmt.Sprintf("Error encode nas security mode complete message: %+v", err))
	}

	if _, err := u.ranConn.Write(encodedNasSecurityModeCompleteMessage); err != nil {
		return errors.New(fmt.Sprintf("Error send nas security mode complete message: %+v", err))
	}
	u.logger.Debug("NAS", fmt.Sprintf("Send NAS Security Mode Complete Message to RAN"))

	// send nas registration complete message to RAN
	nasRegistrationCompleteMessage, err := getNasRegistrationCompleteMessage(nil)
	if err != nil {
		return errors.New(fmt.Sprintf("Error get nas registration complete message: %+v", err))
	}
	encodedNasRegistrationCompleteMessage, err := encodeNasPduWithSecurity(nasRegistrationCompleteMessage, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, u, true, false)
	if err != nil {
		return errors.New(fmt.Sprintf("Error encode nas registration complete message: %+v", err))
	}

	if _, err := u.ranConn.Write(encodedNasRegistrationCompleteMessage); err != nil {
		return errors.New(fmt.Sprintf("Error send nas registration complete message: %+v", err))
	}
	u.logger.Debug("NAS", fmt.Sprintf("Send NAS Registration Complete Message to RAN"))

	u.logger.Info("NAS", fmt.Sprintf("UE Registration finished"))
	return nil
}

func (u *Ue) getBearerType() uint8 {
	if u.accessType == models.AccessType__3_GPP_ACCESS {
		return security.Bearer3GPP
	} else if u.accessType == models.AccessType_NON_3_GPP_ACCESS {
		return security.BearerNon3GPP
	} else {
		return security.OnlyOneBearer
	}
}

func (u *Ue) get5GmmCapability() *nasType.Capability5GMM {
	return &nasType.Capability5GMM{
		Iei:   nasMessage.RegistrationRequestCapability5GMMType,
		Len:   1,
		Octet: [13]uint8{0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
}
