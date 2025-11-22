package ue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Alonza0314/free-ran-ue/constant"
	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/util"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/openapi/models"
	"github.com/songgao/water"
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

type pduSession struct {
	dnn    string
	sNssai *models.Snssai
}

type pduSessionEstablishmentAccept struct {
	ueIp    string
	qosRule []uint8
	dnn     string
	sst     uint8
	sd      [3]uint8
}

type dcRanDataPlane struct {
	ip   string
	port int
}

type nrdc struct {
	enable bool
	dcRanDataPlane
	dcLocalDataPlaneIp string
	specifiedFlow      []string
	rwLock             sync.RWMutex
}

type Ue struct {
	ranControlPlaneIp string
	ranDataPlaneIp    string
	localDataPlaneIp  string

	ranControlPlanePort int
	ranDataPlanePort    int

	ranControlPlaneConn net.Conn
	ranDataPlaneConn    net.Conn
	dcRanDataPlaneConn  net.Conn

	mcc  string
	mnc  string
	msin string

	authentication

	accessType models.AccessType
	authenticationSubscription

	pduSession

	nrdc

	ueTunnelDeviceName string
	ueTunnelDevice     *water.Interface

	readFromTun chan []byte
	readFromRan chan []byte

	pduSessionEstablishmentAccept

	*logger.UeLogger
}

func NewUe(config *model.UeConfig, logger *logger.UeLogger) *Ue {
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

	sstInt, err := strconv.Atoi(config.Ue.PduSession.Snssai.Sst)
	if err != nil {
		logger.CfgLog.Errorf("Error converting sst to int: %v", err)
	}

	return &Ue{
		ranControlPlaneIp: config.Ue.RanControlPlaneIp,
		ranDataPlaneIp:    config.Ue.RanDataPlaneIp,
		localDataPlaneIp:  config.Ue.LocalDataPlaneIp,

		ranControlPlanePort: config.Ue.RanControlPlanePort,
		ranDataPlanePort:    config.Ue.RanDataPlanePort,

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

		pduSession: pduSession{
			dnn: config.Ue.PduSession.Dnn,
			sNssai: &models.Snssai{
				Sst: int32(sstInt),
				Sd:  config.Ue.PduSession.Snssai.Sd,
			},
		},

		nrdc: nrdc{
			enable: config.Ue.Nrdc.Enable,
			dcRanDataPlane: dcRanDataPlane{
				ip:   config.Ue.Nrdc.DcRanDataPlane.Ip,
				port: config.Ue.Nrdc.DcRanDataPlane.Port,
			},
			dcLocalDataPlaneIp: config.Ue.Nrdc.DcLocalDataPlaneIp,
			specifiedFlow:      make([]string, 0),
			rwLock:             sync.RWMutex{},
		},

		ueTunnelDeviceName: config.Ue.UeTunnelDevice,

		UeLogger: logger,
	}
}

func (u *Ue) Start(ctx context.Context, wg *sync.WaitGroup) error {
	u.UeLog.Infof("Starting UE: imsi-%s", u.supi)

	if err := u.connectToRanControlPlane(); err != nil {
		u.UeLog.Errorf("Error connecting to RAN: %v", err)
		return err
	}

	if err := u.processUeRegistration(); err != nil {
		u.UeLog.Errorf("Error processing UE registration: %v", err)
		if err := u.ranControlPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing RAN connection: %v", err)
		}
		return err
	}
	time.Sleep(1 * time.Second)

	if err := u.processPduSessionEstablishment(); err != nil {
		u.UeLog.Errorf("Error processing PDU session establishment: %v", err)
		if err := u.ranControlPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing RAN connection: %v", err)
		}
		return err
	}
	time.Sleep(1 * time.Second)

	if err := u.connectToRanDataPlane(); err != nil {
		u.UeLog.Errorf("Error connecting to RAN data plane: %v", err)
		if err := u.ranControlPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing RAN connection: %v", err)
		}
		return err
	}

	if err := u.setupTunnelDevice(); err != nil {
		u.UeLog.Errorf("Error setting up tunnel device: %v", err)
		if err := u.ranDataPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing RAN connection: %v", err)
		}
		if err := u.ranControlPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing RAN connection: %v", err)
		}
		return err
	}

	// wait for RAN message
	go u.waitForRanMessage(ctx, wg)

	// handle data plane
	go u.handleDataPlane(ctx, wg)

	u.UeLog.Infoln("UE started")
	return nil
}

func (u *Ue) Stop() {
	u.UeLog.Infof("Stopping UE: imsi-%s", u.supi)

	if err := u.processUeDeregistration(); err != nil {
		u.UeLog.Errorf("Error processing UE deregistration: %v", err)
	}

	close(u.readFromTun)
	close(u.readFromRan)

	if err := u.cleanUpTunnelDevice(); err != nil {
		u.UeLog.Errorf("Error cleaning up tunnel device: %v", err)
	}

	if err := u.ranDataPlaneConn.Close(); err != nil {
		u.UeLog.Errorf("Error closing RAN connection: %v", err)
	}

	if u.isNrdcEnabled() {
		if err := u.dcRanDataPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing DC RAN connection: %v", err)
		}
	}

	if err := u.ranControlPlaneConn.Close(); err != nil {
		u.UeLog.Errorf("Error closing RAN connection: %v", err)
	}

	u.UeLog.Infoln("UE stopped")
}

func (u *Ue) connectToRanControlPlane() error {
	u.RanLog.Infoln("Connecting to RAN control plane")

	u.RanLog.Tracef("RAN control plane address: %s:%d", u.ranControlPlaneIp, u.ranControlPlanePort)

	conn, err := util.TcpDialWithOptionalLocalAddress(u.ranControlPlaneIp, u.ranControlPlanePort, "")
	if err != nil {
		return err
	}

	u.RanLog.Debugln("Dial TCP to RAN control plane success")

	u.ranControlPlaneConn = conn

	u.RanLog.Infof("Connected to RAN control plane: %s:%d", u.ranControlPlaneIp, u.ranControlPlanePort)
	return nil
}

func (u *Ue) connectToRanDataPlane() error {
	u.RanLog.Infoln("Connecting to RAN data plane")

	u.RanLog.Tracef("RAN data plane address: %s:%d", u.ranDataPlaneIp, u.ranDataPlanePort)

	conn, err := util.UdpDialWithOptionalLocalAddress(u.ranDataPlaneIp, u.ranDataPlanePort, u.localDataPlaneIp)
	if err != nil {
		return err
	}
	u.ranDataPlaneConn = conn
	u.RanLog.Debugln("Dial UDP to RAN data plane success")

	_, err = u.ranDataPlaneConn.Write([]byte(constant.UE_DATA_PLANE_INITIAL_PACKET + " " + constant.UE_IMSI_PREFIX + u.supi))
	if err != nil {
		return fmt.Errorf("error send initial packet: %+v", err)
	}
	u.RanLog.Debugln("Sent initial packet to RAN data plane UDP server")

	if u.isNrdcEnabled() {
		conn, err := util.UdpDialWithOptionalLocalAddress(u.nrdc.dcRanDataPlane.ip, u.nrdc.dcRanDataPlane.port, u.nrdc.dcLocalDataPlaneIp)
		if err != nil {
			return err
		}
		u.dcRanDataPlaneConn = conn
		u.RanLog.Debugln("Dial UDP to DC RAN data plane success")

		_, err = u.dcRanDataPlaneConn.Write([]byte(constant.UE_DATA_PLANE_INITIAL_PACKET + " " + constant.UE_IMSI_PREFIX + u.supi))
		if err != nil {
			return fmt.Errorf("error send initial packet: %+v", err)
		}
		u.RanLog.Debugln("Sent initial packet to DC RAN data plane UDP server")
	}

	u.RanLog.Infof("Connected to RAN data plane: %s:%d", u.ranDataPlaneIp, u.ranDataPlanePort)
	return nil
}

func (u *Ue) processUeRegistration() error {
	u.RanLog.Infoln("Processing UE Registration")

	mobileIdentity5GS := buildUeMobileIdentity5GS(u.supi)
	u.NasLog.Tracef("Mobile identity 5GS: %+v", mobileIdentity5GS)

	ueSecurityCapability := buildUeSecurityCapability(u.cipheringAlgorithm, u.integrityAlgorithm)
	u.NasLog.Tracef("UE security capability: %+v", ueSecurityCapability)

	// send ue registration request
	registrationRequest, err := getUeRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration, &mobileIdentity5GS, nil, &ueSecurityCapability, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("error get ue registration request: %+v", err)
	}
	u.NasLog.Tracef("Get UE %s registration request: %+v", u.supi, registrationRequest)

	n, err := u.ranControlPlaneConn.Write(registrationRequest)
	if err != nil {
		return fmt.Errorf("error send ue registration request: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of UE %s registration request", n, u.supi)
	u.NasLog.Debugln("Send UE registration request")

	// receive nas authentication request
	nasAuthenticationRequestRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(nasAuthenticationRequestRaw)
	if err != nil {
		return fmt.Errorf("error read nas authentication request: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of NAS Authentication Request from RAN", n)

	nasPdu, err := nasDecode(u, nas.GetSecurityHeaderType(nasAuthenticationRequestRaw[:n]), nasAuthenticationRequestRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas authentication request: %+v", err)
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeAuthenticationRequest {
		return fmt.Errorf("error nas pdu message type: %+v, expected authenticatoin request", nasPdu)
	}
	u.NasLog.Tracef("NAS authentication request: %+v", nasPdu)
	u.NasLog.Debugln("Receive NAS Authentication Request from RAN")

	// calculate for RES* and send nas authentication response
	rand, autn := nasPdu.AuthenticationRequest.GetRANDValue(), nasPdu.AuthenticationRequest.GetAUTN()
	kAmf, kenc, kint, resStar, newSqn, err := deriveResStarAndSetKey(fmt.Sprintf("supi-%s", u.supi), u.cipheringAlgorithm, u.integrityAlgorithm, u.authenticationSubscription.sequenceNumber, u.authenticationSubscription.authenticationManagementField, u.authenticationSubscription.encPermanentKey, u.authenticationSubscription.encOpcKey, rand[:], autn[:], "5G:mnc093.mcc208.3gppnetwork.org")
	if err != nil {
		return fmt.Errorf("error derive res star and set key: %+v", err)
	} else {
		u.kAmf = kAmf
		copy(u.kNasEnc[:], kenc[16:32])
		copy(u.kNasInt[:], kint[16:32])
		u.authenticationSubscription.sequenceNumber = newSqn

		u.NasLog.Tracef("RES*: %+v", resStar)
		u.NasLog.Tracef("kAMF: %+v", kAmf)
		u.NasLog.Tracef("kNAS_ENC: %+v", kenc)
		u.NasLog.Tracef("kNAS_INT: %+v", kint)
		u.NasLog.Tracef("New SQN: %s", newSqn)
	}

	authenticationResponse, err := getAuthenticationResponse(resStar)
	if err != nil {
		return fmt.Errorf("error get authentication response: %+v", err)
	}
	u.NasLog.Tracef("Authentication response: %+v", authenticationResponse)

	n, err = u.ranControlPlaneConn.Write(authenticationResponse)
	if err != nil {
		return fmt.Errorf("error send authentication response: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of Authentication Response to RAN", n)
	u.NasLog.Debugln("Send Authentication Response to RAN")

	// receive nas security mode command message
	nasSecurityCommandRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(nasSecurityCommandRaw)
	if err != nil {
		return fmt.Errorf("error read nas security command: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of NAS Security Mode Command from RAN", n)

	nasPdu, err = nasDecode(u, nas.GetSecurityHeaderType(nasSecurityCommandRaw[:n]), nasSecurityCommandRaw[:n])
	if err != nil {
		return fmt.Errorf("error get nas pdu: %+v", err)
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeSecurityModeCommand {
		return fmt.Errorf("error nas pdu message type: %+v, expected security mode command", nasPdu)
	}
	u.NasLog.Tracef("NAS security mode command: %+v", nasPdu)
	u.NasLog.Debugln("Receive NAS Security Mode Command from RAN")

	// send nas security mode complete message
	registrationRequestWith5Gmm, err := getUeRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration, &mobileIdentity5GS, nil, &ueSecurityCapability, u.get5GmmCapability(), nil, nil)
	if err != nil {
		return fmt.Errorf("error get ue registration request with 5GMM: %+v", err)
	}
	u.NasLog.Tracef("Registration request with 5GMM: %+v", registrationRequestWith5Gmm)

	nasSecurityModeCompleteMessage, err := getNasSecurityModeCompleteMessage(registrationRequestWith5Gmm)
	if err != nil {
		return fmt.Errorf("error get nas security mode complete message: %+v", err)
	}
	u.NasLog.Tracef("NAS security mode complete message: %+v", nasSecurityModeCompleteMessage)

	encodedNasSecurityModeCompleteMessage, err := encodeNasPduWithSecurity(nasSecurityModeCompleteMessage, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, u, true, true)
	if err != nil {
		return fmt.Errorf("error encode nas security mode complete message: %+v", err)
	}
	u.NasLog.Tracef("Encoded NAS security mode complete message: %+v", encodedNasSecurityModeCompleteMessage)

	n, err = u.ranControlPlaneConn.Write(encodedNasSecurityModeCompleteMessage)
	if err != nil {
		return fmt.Errorf("error send nas security mode complete message: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of NAS Security Mode Complete Message to RAN", n)
	u.NasLog.Debugln("Send NAS Security Mode Complete Message to RAN")

	// receive nas registration accept
	nasRegistrationAcceptRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(nasRegistrationAcceptRaw)
	if err != nil {
		return fmt.Errorf("error read nas registration accept: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of NAS Registration Accept from RAN", n)

	nasPdu, err = nasDecode(u, nas.GetSecurityHeaderType(nasRegistrationAcceptRaw[:n]), nasRegistrationAcceptRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas registration accept: %+v", err)
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeRegistrationAccept {
		return fmt.Errorf("error nas pdu message type: %+v, expected registration accept", nasPdu)
	}
	u.NasLog.Tracef("NAS registration accept: %+v", nasPdu)
	u.NasLog.Debugln("Receive NAS Registration Accept from RAN")

	// send nas registration complete message to RAN
	nasRegistrationCompleteMessage, err := getNasRegistrationCompleteMessage(nil)
	if err != nil {
		return fmt.Errorf("error get nas registration complete message: %+v", err)
	}
	u.NasLog.Tracef("NAS registration complete message: %+v", nasRegistrationCompleteMessage)

	encodedNasRegistrationCompleteMessage, err := encodeNasPduWithSecurity(nasRegistrationCompleteMessage, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, u, true, false)
	if err != nil {
		return fmt.Errorf("error encode nas registration complete message: %+v", err)
	}
	u.NasLog.Tracef("Encoded NAS registration complete message: %+v", encodedNasRegistrationCompleteMessage)

	n, err = u.ranControlPlaneConn.Write(encodedNasRegistrationCompleteMessage)
	if err != nil {
		return fmt.Errorf("error send nas registration complete message: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of NAS Registration Complete Message to RAN", n)
	u.NasLog.Debugln("Send NAS Registration Complete Message to RAN")

	// receive nas configuration update command
	nasConfigurationUpdateCommandRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(nasConfigurationUpdateCommandRaw)
	if err != nil {
		return fmt.Errorf("error read nas configuration update command: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of NAS Configuration Update Command from RAN", n)

	nasPdu, err = nasDecode(u, nas.GetSecurityHeaderType(nasConfigurationUpdateCommandRaw[:n]), nasConfigurationUpdateCommandRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas configuration update command: %+v", err)
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeConfigurationUpdateCommand {
		return fmt.Errorf("error nas pdu message type: %+v, expected configuration update command", nasPdu)
	}
	u.NasLog.Tracef("NAS configuration update command: %+v", nasPdu)
	u.NasLog.Debugln("Receive NAS Configuration Update Command from RAN")

	u.RanLog.Infoln("UE Registration finished")
	return nil
}

func (u *Ue) processPduSessionEstablishment() error {
	u.PduLog.Infoln("Processing PDU session establishment")

	// send pdu session establishment request
	pduSessionEstablishmentRequest, err := getPduSessionEstablishmentRequest(constant.PDU_SESSION_ID)
	if err != nil {
		return fmt.Errorf("error get pdu session establishment request: %+v", err)
	}
	u.NasLog.Tracef("PDU session establishment request: %+v", pduSessionEstablishmentRequest)

	ulNasTransportPduSessionEstablishmentRequest, err := getUlNasTransportMessage(pduSessionEstablishmentRequest, constant.PDU_SESSION_ID, nasMessage.ULNASTransportRequestTypeInitialRequest, u.pduSession.dnn, u.pduSession.sNssai)
	if err != nil {
		return fmt.Errorf("error get ul nas transport pdu session establishment request: %+v", err)
	}
	u.NasLog.Tracef("UL NAS transport pdu session establishment request: %+v", ulNasTransportPduSessionEstablishmentRequest)

	encodedUlNasTransportPduSessionEstablishmentRequest, err := encodeNasPduWithSecurity(ulNasTransportPduSessionEstablishmentRequest, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, u, true, false)
	if err != nil {
		return fmt.Errorf("error encode ul nas transport pdu session establishment request: %+v", err)
	}
	u.NasLog.Tracef("Encoded UL NAS transport pdu session establishment request: %+v", encodedUlNasTransportPduSessionEstablishmentRequest)

	n, err := u.ranControlPlaneConn.Write(encodedUlNasTransportPduSessionEstablishmentRequest)
	if err != nil {
		return fmt.Errorf("error send ul nas transport pdu session establishment request: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of UL NAS transport pdu session establishment request to RAN", n)
	u.NasLog.Debugln("Send UL NAS transport pdu session establishment request to RAN")

	// receive pdu session establishment accept
	nasPduSessionEstablishmentAcceptRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(nasPduSessionEstablishmentAcceptRaw)
	if err != nil {
		return fmt.Errorf("error read nas pdu session establishment accept: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of NAS PDU Session Establishment Accept from RAN", n)

	nasPduSessionEstablishmentAccept, err := nasDecode(u, nas.GetSecurityHeaderType(nasPduSessionEstablishmentAcceptRaw[:n]), nasPduSessionEstablishmentAcceptRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode nas pdu session establishment accept: %+v", err)
	}
	if nasPduSessionEstablishmentAccept.GmmHeader.GetMessageType() != nas.MsgTypeDLNASTransport {
		return fmt.Errorf("error nas pdu message type: %+v, expected pdu session establishment accept", nasPduSessionEstablishmentAccept.GmmHeader.GetMessageType())
	}
	u.NasLog.Tracef("NAS PDU Session Establishment Accept: %+v", nasPduSessionEstablishmentAccept)
	u.NasLog.Debugln("Receive NAS PDU Session Establishment Accept from RAN")

	// store ue information
	if err := u.extractUeInformationFromNasPduSessionEstablishmentAccept(nasPduSessionEstablishmentAccept); err != nil {
		return fmt.Errorf("error extract ue information from nas pdu session establishment accept: %+v", err)
	}

	u.PduLog.Infof("UE %s PDU session establishment complete", u.supi)
	return nil
}

func (u *Ue) processUeDeregistration() error {
	u.RanLog.Infoln("Processing UE deregistration")

	mobileIdentity5GS := buildUeMobileIdentity5GS(u.supi)
	u.NasLog.Tracef("Mobile identity 5GS: %+v", mobileIdentity5GS)

	// send ue deregistration request
	deregistrationRequest, err := getUeDeRegistrationRequest(nasMessage.AccessType3GPP, 0x00, 0x04, mobileIdentity5GS)
	if err != nil {
		return fmt.Errorf("error get ue deregistration request: %+v", err)
	}
	u.NasLog.Tracef("Get UE deregistration request: %+v", deregistrationRequest)

	encodedDeregistrationRequest, err := encodeNasPduWithSecurity(deregistrationRequest, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, u, true, false)
	if err != nil {
		return fmt.Errorf("error encode ue deregistration request: %+v", err)
	}
	u.NasLog.Tracef("Encoded UE deregistration request: %+v", encodedDeregistrationRequest)

	n, err := u.ranControlPlaneConn.Write(encodedDeregistrationRequest)
	if err != nil {
		return fmt.Errorf("error send ue deregistration request: %+v", err)
	}
	u.NasLog.Tracef("Sent %d bytes of UE deregistration request to RAN", n)
	u.NasLog.Debugln("Send UE deregistration request to RAN")

	// receive ue deregistration accept
	ueDeRegistrationAcceptRaw := make([]byte, 1024)
	n, err = u.ranControlPlaneConn.Read(ueDeRegistrationAcceptRaw)
	if err != nil {
		return fmt.Errorf("error read ue deregistration accept: %+v", err)
	}
	u.NasLog.Tracef("Received %d bytes of UE deregistration accept from RAN", n)

	ueDeRegistrationAccept, err := nasDecode(u, nas.GetSecurityHeaderType(ueDeRegistrationAcceptRaw[:n]), ueDeRegistrationAcceptRaw[:n])
	if err != nil {
		return fmt.Errorf("error decode ue deregistration accept: %+v", err)
	}
	if ueDeRegistrationAccept.GmmHeader.GetMessageType() != nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration {
		return fmt.Errorf("error nas pdu message type: %+v, expected pdu session establishment accept", ueDeRegistrationAccept.GmmHeader.GetMessageType())
	}
	u.NasLog.Tracef("NAS UE deregistration accept: %+v", ueDeRegistrationAccept)
	u.NasLog.Debugln("Receive NAS UE deregistration accept from RAN")

	u.RanLog.Infoln("UE deregistration complete")
	return nil
}

func (u *Ue) extractUeInformationFromNasPduSessionEstablishmentAccept(nasPduSessionEstablishmentAccept *nas.Message) error {
	nasMessage, err := getNasPduFromNasPduSessionEstablishmentAccept(nasPduSessionEstablishmentAccept)
	if err != nil {
		return fmt.Errorf("error get nas pdu from nas pdu session establishment accept: %+v", err)
	}
	u.NasLog.Tracef("NAS message: %+v", nasMessage)

	switch nasMessage.GsmHeader.GetMessageType() {
	case nas.MsgTypePDUSessionEstablishmentAccept:
		pduSessionEstablishmentAccept := nasMessage.PDUSessionEstablishmentAccept

		pduAddress := pduSessionEstablishmentAccept.GetPDUAddressInformation()
		u.pduSessionEstablishmentAccept.ueIp = fmt.Sprintf("%d.%d.%d.%d", pduAddress[0], pduAddress[1], pduAddress[2], pduAddress[3])
		u.PduLog.Infof("PDU session UE IP: %s", u.pduSessionEstablishmentAccept.ueIp)

		u.pduSessionEstablishmentAccept.qosRule = pduSessionEstablishmentAccept.AuthorizedQosRules.GetQosRule()
		u.nrdc.specifiedFlow = append(u.nrdc.specifiedFlow, util.GetQosRule(u.pduSessionEstablishmentAccept.qosRule, u.UeLogger)...)
		u.PduLog.Infof("PDU session QoS rule: %+v", u.nrdc.specifiedFlow)

		u.pduSessionEstablishmentAccept.dnn = pduSessionEstablishmentAccept.GetDNN()
		u.PduLog.Infof("PDU session DNN: %s", u.pduSessionEstablishmentAccept.dnn)

		u.pduSessionEstablishmentAccept.sst = pduSessionEstablishmentAccept.GetSST()
		u.pduSessionEstablishmentAccept.sd = pduSessionEstablishmentAccept.GetSD()
		u.PduLog.Infof("PDU session SNSSAI, sst: %d, sd: %s", u.pduSessionEstablishmentAccept.sst, fmt.Sprintf("%x%x%x", u.pduSessionEstablishmentAccept.sd[0], u.pduSessionEstablishmentAccept.sd[1], u.pduSessionEstablishmentAccept.sd[2]))
	case nas.MsgTypePDUSessionReleaseCommand:
		return fmt.Errorf("not implemented: PDUSessionReleaseCommand")
	case nas.MsgTypePDUSessionEstablishmentReject:
		return fmt.Errorf("not implemented: PDUSessionEstablishmentReject")
	default:
		return fmt.Errorf("not implemented: %+v", nasMessage.GsmHeader.GetMessageType())
	}

	return nil
}

func (u *Ue) waitForRanMessage(ctx context.Context, wg *sync.WaitGroup) {
	u.RanLog.Infoln("Waiting for RAN message")
	wg.Add(1)

	buffer := make([]byte, 1024)
	for {
		if err := u.ranControlPlaneConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			u.RanLog.Errorf("Error set read deadline: %+v", err)
			goto STOP_WAITING
		}
		select {
		case <-ctx.Done():
			if err := u.ranControlPlaneConn.SetReadDeadline(time.Time{}); err != nil {
				u.RanLog.Errorf("Error set read deadline: %+v", err)
			}
			goto STOP_WAITING
		default:
			n, err := u.ranControlPlaneConn.Read(buffer)
			if err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					goto STOP_WAITING
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				u.RanLog.Warnf("Error read from ran control plane: %+v", err)
			}

			switch string(buffer[:n]) {
			case constant.UE_TUNNEL_UPDATE:
				go u.updateDataPlane()
			default:
				u.RanLog.Warnf("Received unknown message from RAN: %+v", buffer[:n])
			}
		}
	}
STOP_WAITING:
	u.RanLog.Infoln("Stop waiting for RAN message")
	wg.Done()
}

func (u *Ue) setupTunnelDevice() error {
	u.TunLog.Infoln("Setting up UE tunnel device")

	waterInterface, err := bringUpUeTunnelDevice(u.ueTunnelDeviceName, u.ueIp)
	if err != nil {
		return fmt.Errorf("error bring up ue tunnel device: %+v", err)
	}
	u.TunLog.Debugln("Bring up ue tunnel device success")

	u.ueTunnelDevice = waterInterface

	// go routine for read data from TUN
	u.readFromTun = make(chan []byte)
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := u.ueTunnelDevice.Read(buffer)
			if err != nil {
				u.TunLog.Errorf("Error read from ue tunnel device: %+v", err)
				return
			}
			version := buffer[0] >> 4
			if version == 6 {
				continue
			}

			tmp := make([]byte, n)
			copy(tmp, buffer[:n])
			u.readFromTun <- tmp
		}
	}()
	u.TunLog.Debugln("Read from TUN started")

	// go routing for read data from RAN
	u.readFromRan = make(chan []byte, 2)
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := u.ranDataPlaneConn.Read(buffer)
			if err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					u.TunLog.Debugln("RAN data plane connection closed")
					return
				}
				u.RanLog.Errorf("Error read from ran data plane: %+v", err)
				return
			}

			tmp := make([]byte, n)
			copy(tmp, buffer[:n])
			u.readFromRan <- tmp
		}
	}()
	u.TunLog.Debugln("Read from RAN started")

	if u.isNrdcEnabled() {
		go func() {
			buffer := make([]byte, 4096)
			for {
				n, err := u.dcRanDataPlaneConn.Read(buffer)
				if err != nil {
					if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
						u.TunLog.Debugln("DC RAN data plane connection closed")
						return
					}
					u.RanLog.Errorf("Error read from dc ran data plane: %+v", err)
				}

				tmp := make([]byte, n)
				copy(tmp, buffer[:n])
				u.readFromRan <- tmp
			}
		}()
		u.TunLog.Debugln("Read from DC RAN data plane started")
	}

	u.TunLog.Infof("UE tunnel device setup as %s", u.ueTunnelDeviceName)
	return nil
}

func (u *Ue) cleanUpTunnelDevice() error {
	u.TunLog.Infoln("Cleaning up UE tunnel device")

	if err := bringDownUeTunnelDevice(u.ueTunnelDeviceName); err != nil {
		return fmt.Errorf("error bring down ue tunnel device: %+v", err)
	}
	u.TunLog.Debugln("Bring down ue tunnel device success")

	u.TunLog.Infoln("UE tunnel device cleaned up")
	return nil
}

func (u *Ue) handleDataPlane(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)

	// forward data from TUN to RAN and RAN to TUN
	for {
		select {
		case <-ctx.Done():
			goto HANDLE_DATA_PLANE_FINISH
		case buffer := <-u.readFromTun:
			if !u.isNrdcEnabled() {
				n, err := u.ranDataPlaneConn.Write(buffer)
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						goto HANDLE_DATA_PLANE_FINISH
					}
					u.RanLog.Warnf("Error sent to ran data plane: %+v", err)
				}
				u.RanLog.Tracef("Sent %d bytes of data to RAN: %+v", n, buffer[:n])
			} else {
				if util.IsIpInSpecifiedFlow(buffer, u.nrdc.specifiedFlow) {
					n, err := u.dcRanDataPlaneConn.Write(buffer)
					if err != nil {
						if errors.Is(err, net.ErrClosed) {
							goto HANDLE_DATA_PLANE_FINISH
						}
						u.RanLog.Warnf("Error sent to dc ran data plane: %+v", err)
					}
					u.RanLog.Tracef("Sent %d bytes of data to DC RAN: %+v", n, buffer[:n])
				} else {
					n, err := u.ranDataPlaneConn.Write(buffer)
					if err != nil {
						if errors.Is(err, net.ErrClosed) {
							goto HANDLE_DATA_PLANE_FINISH
						}
						u.RanLog.Warnf("Error sent to ran data plane: %+v", err)
					}
					u.RanLog.Tracef("Sent %d bytes of data to RAN: %+v", n, buffer[:n])
				}
			}
		case buffer := <-u.readFromRan:
			n, err := u.ueTunnelDevice.Write(buffer)
			if err != nil {
				u.TunLog.Warnf("Error write to ue tunnel device: %+v", err)
			}
			u.TunLog.Tracef("Wrote %d bytes of data to TUN: %+v", n, buffer[:n])
		}
	}

HANDLE_DATA_PLANE_FINISH:
	wg.Done()
}

func (u *Ue) updateDataPlane() {
	u.TunLog.Infoln("Updating data plane")

	u.rwLock.Lock()
	defer u.rwLock.Unlock()

	if !u.nrdc.enable {
		conn, err := util.UdpDialWithOptionalLocalAddress(u.nrdc.dcRanDataPlane.ip, u.nrdc.dcRanDataPlane.port, u.nrdc.dcLocalDataPlaneIp)
		if err != nil {
			u.TunLog.Errorf("Error connect to dc ran data plane: %+v", err)
			return
		}
		u.dcRanDataPlaneConn = conn

		_, err = u.dcRanDataPlaneConn.Write([]byte(constant.UE_DATA_PLANE_INITIAL_PACKET + " " + constant.UE_IMSI_PREFIX + u.supi))
		if err != nil {
			u.TunLog.Errorf("Error send initial packet: %+v", err)
			return
		}
		u.RanLog.Debugln("Sent initial packet to DC RAN data plane UDP server")

		go func() {
			buffer := make([]byte, 4096)
			for {
				n, err := u.dcRanDataPlaneConn.Read(buffer)
				if err != nil {
					if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
						u.TunLog.Debugln("DC RAN data plane connection closed")
						return
					}
					u.RanLog.Errorf("Error read from dc ran data plane: %+v", err)
				}

				tmp := make([]byte, n)
				copy(tmp, buffer[:n])
				u.readFromRan <- tmp
			}
		}()
		u.TunLog.Debugln("Read from DC RAN data plane started")

		u.nrdc.enable = true
		u.TunLog.Infoln("Data plane is updated to NRDC mode")
	} else {
		if err := u.dcRanDataPlaneConn.Close(); err != nil {
			u.UeLog.Errorf("Error closing DC RAN connection: %v", err)
		}

		u.nrdc.enable = false
		u.TunLog.Infoln("Data plane is updated to non-NRDC mode")
	}
}

func (u *Ue) getBearerType() uint8 {
	switch u.accessType {
	case models.AccessType__3_GPP_ACCESS:
		return security.Bearer3GPP
	case models.AccessType_NON_3_GPP_ACCESS:
		return security.BearerNon3GPP
	default:
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

func (u *Ue) isNrdcEnabled() bool {
	u.rwLock.RLock()
	defer u.rwLock.RUnlock()

	return u.nrdc.enable
}
