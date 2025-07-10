package gnb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/util"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/sctp"
)

type Gnb struct {
	amfN2Ip string
	gnbN2Ip string
	ranIp   string

	amfN2Port int
	gnbN2Port int
	ranPort   int

	gnbULTeid string
	gnbDLTeid string

	n2Conn      *sctp.SCTPConn
	ranListener *net.Listener

	ngapPpid uint32

	gnbId   []byte
	gnbName string

	plmnId ngapType.PLMNIdentity
	tai    ngapType.TAI
	snssai ngapType.SNSSAI

	activeConns sync.Map

	*logger.GnbLogger
}

func NewGnb(config *model.GnbConfig, gnbLogger *logger.GnbLogger) *Gnb {
	gnbId, err := util.HexStringToBytes(config.Gnb.GnbId)
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting gnbId to escaped: %v", err)
		return nil
	}

	plmnId, err := util.PlmnIdToNgap(models.PlmnId{
		Mcc: config.Gnb.PlmnId.Mcc,
		Mnc: config.Gnb.PlmnId.Mnc,
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting plmnId to ngap: %v", err)
		return nil
	}

	tai, err := util.TaiToNgap(models.Tai{
		Tac: config.Gnb.Tai.Tac,
		PlmnId: &models.PlmnId{
			Mcc: config.Gnb.Tai.BroadcastPlmnId.Mcc,
			Mnc: config.Gnb.Tai.BroadcastPlmnId.Mnc,
		},
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting tai to ngap: %v", err)
		return nil
	}

	sstInt, err := strconv.Atoi(config.Gnb.Snssai.Sst)
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting sst to int: %v", err)
		return nil
	}
	snssai, err := util.SNssaiToNgap(models.Snssai{
		Sst: int32(sstInt),
		Sd:  config.Gnb.Snssai.Sd,
	})
	if err != nil {
		gnbLogger.CfgLog.Errorf("Error converting snssai to ngap: %v", err)
		return nil
	}

	return &Gnb{
		amfN2Ip: config.Gnb.AmfN2Ip,
		gnbN2Ip: config.Gnb.GnbN2Ip,
		ranIp:   config.Gnb.RanIp,

		amfN2Port: config.Gnb.AmfN2Port,
		gnbN2Port: config.Gnb.GnbN2Port,
		ranPort:   config.Gnb.RanPort,

		ngapPpid: config.Gnb.NgapPpid,

		gnbId:   gnbId,
		gnbName: config.Gnb.GnbName,

		plmnId: plmnId,
		tai:    tai,
		snssai: snssai,

		GnbLogger: gnbLogger,
	}
}

func (g *Gnb) Start(ctx context.Context) error {
	g.RanLog.Infoln("Starting GNB")
	if err := g.connectToAmf(); err != nil {
		g.SctpLog.Errorf("Error connecting to AMF: %v", err)
		return err
	}

	if err := g.setupN2(); err != nil {
		g.NgapLog.Errorf("Error setting up N2: %v", err)
		return err
	}

	if err := g.startRanListener(); err != nil {
		g.RanLog.Errorf("Error starting gNB listener: %v", err)
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := (*g.ranListener).Accept()
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						return
					}
					g.RanLog.Errorf("Error accepting UE connection: %v", err)
					continue
				}
				g.RanLog.Infof("New UE connection accepted from: %v", conn.RemoteAddr())
				g.activeConns.Store(conn, struct{}{})
				go g.handleRanConnection(ctx, conn)
			}
		}
	}()

	g.RanLog.Infoln("GNB started")
	return nil
}

func (g *Gnb) Stop() {
	g.RanLog.Infoln("Stopping GNB")
	if err := (*g.ranListener).Close(); err != nil {
		g.RanLog.Errorf("Error stopping gNB: %v", err)
		return
	}
	g.RanLog.Debugln("gNB listener stopped")
	g.RanLog.Traceln("gNB listener stopped at %s:%d", g.ranIp, g.ranPort)

	var wg sync.WaitGroup
	g.activeConns.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			if conn, ok := key.(net.Conn); ok {
				g.RanLog.Tracef("UE %v still in connection", conn.RemoteAddr())
				if err := conn.Close(); err != nil {
					g.RanLog.Errorf("Error closing UE connection: %v", err)
				}
			}
			g.RanLog.Debugf("Closed UE connection from: %v", conn.RemoteAddr())
		}(key.(net.Conn))
		return true
	})
	wg.Wait()

	if err := g.n2Conn.Close(); err != nil {
		g.SctpLog.Errorf("Error stopping N2 connection: %v", err)
		return
	}
	g.SctpLog.Debugln("N2 connection closed")
	g.SctpLog.Traceln("N2 connection closed at %s:%d", g.gnbN2Ip, g.gnbN2Port)

	g.RanLog.Infoln("GNB stopped")
}

func (g *Gnb) connectToAmf() error {
	g.RanLog.Infoln("Connecting to AMF")

	amfAddr, gnbAddr, err := getAmfAndGnbSctpN2Addr(g.amfN2Ip, g.gnbN2Ip, g.amfN2Port, g.gnbN2Port)
	if err != nil {
		return err
	}
	g.SctpLog.Tracef("AMF N2 Address: %v", amfAddr.String())
	g.SctpLog.Tracef("GNB N2 Address: %v", gnbAddr.String())

	conn, err := sctp.DialSCTP("sctp", gnbAddr, amfAddr)
	if err != nil {
		return fmt.Errorf("Error connecting to AMF: %v", err)
	}
	g.SctpLog.Debugln("Dail SCTP to AMF success")

	info, err := conn.GetDefaultSentParam()
	if err != nil {
		return err
	}
	g.SctpLog.Tracef("N2 connection default sent param: %+v", info)

	info.PPID = g.ngapPpid
	if err := conn.SetDefaultSentParam(info); err != nil {
		return fmt.Errorf("Error setting default sent param: %v", err)
	}

	g.n2Conn = conn

	g.RanLog.Infof("Connected to AMF: %v", amfAddr.String())
	return nil
}

func (g *Gnb) setupN2() error {
	g.RanLog.Infoln("Setting up N2")

	request, err := getNgapSetupRequest(g.gnbId, g.gnbName, g.plmnId, g.tai, g.snssai)
	if err != nil {
		return fmt.Errorf("Error getting NGAP setup request: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup request: %+v", request)

	n, err := g.n2Conn.Write(request)
	if err != nil {
		return fmt.Errorf("Error sending NGAP setup request: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of NGAP setup request", n)
	g.NgapLog.Debugln("Sent NGAP setup request to AMF")

	responseRaw := make([]byte, 2048)
	n, err = g.n2Conn.Read(responseRaw)
	if err != nil {
		return fmt.Errorf("Error reading NGAP setup response: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup responseRaw: %+v", responseRaw[:n])
	g.NgapLog.Debugln("Received NGAP setup response from AMF")

	response, err := ngap.Decoder(responseRaw[:n])
	if err != nil {
		return fmt.Errorf("Error decoding NGAP setup response: %v", err)
	}
	g.NgapLog.Tracef("NGAP setup response: %+v", response)

	if (response.Present != ngapType.NGAPPDUPresentSuccessfulOutcome) || (response.SuccessfulOutcome.ProcedureCode.Value != ngapType.ProcedureCodeNGSetup) {
		return fmt.Errorf("Error NGAP setup response: %+v", response)
	}

	g.NgapLog.Infoln("============= gNB Info =============")

	gnbId := util.BytesToHexString(g.gnbId)
	g.NgapLog.Infof("gNB ID: %v, name: %s", gnbId, g.gnbName)

	plmnId := ngapConvert.PlmnIdToModels(g.plmnId)
	g.NgapLog.Infof("PLMN ID: %v", plmnId)

	tai := ngapConvert.TaiToModels(g.tai)
	g.NgapLog.Infof("TAC: %v, broadcast PLMN ID: %v", tai.Tac, tai.PlmnId)

	snssai := ngapConvert.SNssaiToModels(g.snssai)
	g.NgapLog.Infof("SST: %v, SD: %v", snssai.Sst, snssai.Sd)

	g.NgapLog.Infoln("====================================")

	g.RanLog.Infoln("N2 setup complete")
	return nil
}

func (g *Gnb) setupN1(n1Conn net.Conn) error {
	g.RanLog.Infoln("Setting up N1")

	// ue initialization
	mobileIdentity5GS, err := g.processUeInitialization(n1Conn)
	if err != nil {
		return err
	}

	g.RanLog.Infof("UE %s N1 setup complete", mobileIdentity5GS.GetSUCI())
	return nil
}

func (g *Gnb) startRanListener() error {
	g.RanLog.Infoln("Starting RAN listener")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.ranIp, g.ranPort))
	if err != nil {
		return err
	}
	g.ranListener = &listener

	g.RanLog.Infoln("============= RAN Info =============")
	g.RanLog.Infof("RAN access address: %s:%d", g.ranIp, g.ranPort)
	g.RanLog.Infoln("====================================")

	g.RanLog.Infoln("RAN listener started")
	return nil
}

func (g *Gnb) handleRanConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			g.RanLog.Errorf("Error closing UE connection: %v", err)
		}
		g.RanLog.Infof("Closed UE connection from: %v", conn.RemoteAddr())
		g.activeConns.Delete(conn)
	}()

	if err := g.setupN1(conn); err != nil {
		g.RanLog.Errorf("Error setting up N1: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, 1024)
			_, err := conn.Read(buffer)
			if err != nil {
				if err == io.EOF {
					g.RanLog.Debugf("UE connection closed by client: %v", conn.RemoteAddr())
					return
				}
				g.RanLog.Errorf("Error reading from UE connection: %v", err)
				return
			}
		}
	}
}

func (g *Gnb) processUeInitialization(n1Conn net.Conn) (nasType.MobileIdentity5GS, error) {
	g.RanLog.Infoln("Processing UE initialization")

	var mobileIdentity5GS nasType.MobileIdentity5GS

	// receive ue registration request from UE and send to AMF
	ueRegistrationRequest := make([]byte, 1024)
	n, err := n1Conn.Read(ueRegistrationRequest)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error receive ue registration request from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of UE registration request from UE", n)

	nasMessage := nas.NewMessage()
	if err := nasMessage.GmmMessageDecode(&ueRegistrationRequest); err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error decode ue registration request from UE: %v", err)
	}
	mobileIdentity5GS = nasMessage.GmmMessage.RegistrationRequest.MobileIdentity5GS
	g.NasLog.Debugf("Receive UE %s registration request from UE", mobileIdentity5GS.GetSUCI())

	ueInitialMessage, err := getInitialUeMessage(1, ueRegistrationRequest, g.plmnId, g.tai)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error get initial ue message: %v", err)
	}
	g.NgapLog.Tracef("Get initial UE message: %+v", ueInitialMessage)

	if n, err = g.n2Conn.Write(ueInitialMessage); err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send initial ue message to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of initial UE message to AMF", n)
	g.NgapLog.Debugln("Sent initial UE message to AMF")

	// receive nas authentication request from AMF and send to UE
	nasAuthenticationRequest := make([]byte, 1024)
	n, err = g.n2Conn.Read(nasAuthenticationRequest)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error receive initial ue response from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NAS Authentication Request from AMF", n)
	g.NgapLog.Debugln("Receive NAS Authentication Request from AMF")

	if n, err = n1Conn.Write(nasAuthenticationRequest[:n]); err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send nas authentication request to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS Authentication Request to UE", n)
	g.NasLog.Debugln("Send NAS Authentication Request to UE")

	// receive nas authentication response from UE and send to AMF
	nasAuthenticationResponse := make([]byte, 1024)
	n, err = n1Conn.Read(nasAuthenticationResponse)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error receive nas authentication response from UE: %v", err)
	}
	g.NasLog.Tracef("Received %d bytes of NAS Authentication Response from UE", n)
	g.NasLog.Debugln("Receive NAS Authentication Response from UE")

	uplinkNasTransport, err := getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasAuthenticationResponse[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Sent uplink NAS transport to AMF")

	// receive nas security mode command message from AMF and send to UE
	nasSecurityModeCommand := make([]byte, 1024)
	n, err = g.n2Conn.Read(nasSecurityModeCommand)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error receive nas security mode command from AMF: %v", err)
	}
	g.NgapLog.Tracef("Received %d bytes of NAS Security Mode Command from AMF", n)
	g.NgapLog.Debugf("Receive NAS Security Mode Command from AMF")

	if n, err = n1Conn.Write(nasSecurityModeCommand[:n]); err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send nas security mode command to UE: %v", err)
	}
	g.NasLog.Tracef("Sent %d bytes of NAS Security Mode Command to UE", n)
	g.NasLog.Debugln("Send NAS Security Mode Command to UE")

	// receive nas security mode complete message from UE and send to AMF
	nasSecurityModeComplete := make([]byte, 1024)
	n, err = n1Conn.Read(nasSecurityModeComplete)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas security mode complete from UE: %v", err))
	}
	g.NasLog.Tracef("Received %d bytes of NAS Security Mode Complete from UE", n)
	g.NasLog.Debugln("Receive NAS Security Mode Complete from UE")

	uplinkNasTransport, err = getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasSecurityModeComplete[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send uplink nas transport to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Sent uplink NAS transport to AMF")

	// receive ngap initial context setup request from AMF
	ngapInitialContextSetupRequestRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapInitialContextSetupRequestRaw)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive ngap initial context setup request from AMF: %v", err))
	}
	g.NgapLog.Tracef("Received %d bytes of NGAP Initial Context Setup Request from AMF", n)
	g.NgapLog.Debugln("Receive NGAP Initial Context Setup Request from AMF")

	ngapInitialContextSetupRequest, err := ngap.Decoder(ngapInitialContextSetupRequestRaw[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error decode ngap initial context setup request from AMF: %v", err))
	}
	if ngapInitialContextSetupRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapInitialContextSetupRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeInitialContextSetup {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error ngap initial context setup request: no initial context setup request"))
	}
	g.NgapLog.Tracef("NGAP Initial Context Setup Request: %+v", ngapInitialContextSetupRequest)
	g.NgapLog.Debugln("Receive NGAP Initial Context Setup Request from AMF")

	// send ngap initial context setup response to AMF
	ngapInitialContextSetupResponse, err := getNgapInitialContextSetupResponse(1, 1)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get ngap initial context setup response: %v", err))
	}
	g.NgapLog.Tracef("Get NGAP Initial Context Setup Response: %+v", ngapInitialContextSetupResponse)

	n, err = g.n2Conn.Write(ngapInitialContextSetupResponse)
	if err != nil {
		return mobileIdentity5GS, fmt.Errorf("Error send ngap initial context setup response to AMF: %v", err)
	}
	g.NgapLog.Tracef("Sent %d bytes of NGAP Initial Context Setup Response to AMF", n)
	g.NgapLog.Debugln("Send NGAP Initial Context Setup Response to AMF")

	// receive nas registration complete message from UE and send to AMF
	nasRegistrationComplete := make([]byte, 1024)
	n, err = n1Conn.Read(nasRegistrationComplete)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas registration complete from UE: %v", err))
	}
	g.NasLog.Tracef("Received %d bytes of NAS Registration Complete from UE", n)
	g.NasLog.Debugln("Receive NAS Registration Complete from UE")

	uplinkNasTransport, err = getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasRegistrationComplete[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	g.NgapLog.Tracef("Get uplink NAS transport: %+v", uplinkNasTransport)

	n, err = g.n2Conn.Write(uplinkNasTransport)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send uplink nas transport to AMF: %v", err))
	}
	g.NgapLog.Tracef("Sent %d bytes of uplink NAS transport to AMF", n)
	g.NgapLog.Debugln("Send NAS Registration Complete to AMF")

	// receive ue configuration update command message from AMF
	ueConfigurationUpdateCommandRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ueConfigurationUpdateCommandRaw)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive ue configuration update command from AMF: %v", err))
	}
	g.NgapLog.Tracef("Received %d bytes of UE Configuration Update Command from AMF", n)
	g.NgapLog.Debugln("Receive UE Configuration Update Command from AMF")

	ueConfigurationUpdateCommand, err := ngap.Decoder(ueConfigurationUpdateCommandRaw[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error decode ue configuration update command from AMF: %v", err))
	}
	if ueConfigurationUpdateCommand.Present != ngapType.NGAPPDUPresentInitiatingMessage || ueConfigurationUpdateCommand.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error ue configuration update command: no ue configuration update command"))
	}
	g.NgapLog.Tracef("UE Configuration Update Command: %+v", ueConfigurationUpdateCommand)
	g.NgapLog.Debugln("Receive UE Configuration Update Command from AMF")

	g.RanLog.Infof("UE %s initialized", mobileIdentity5GS.GetSUCI())
	return mobileIdentity5GS, nil
}
