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

	logger *logger.Logger
}

func NewGnb(config *model.GnbConfig, logger *logger.Logger) *Gnb {
	gnbId, err := util.HexStringToBytes(config.Gnb.GnbId)
	if err != nil {
		logger.Error("CONFIG", fmt.Sprintf("Error converting gnbId to escaped: %v", err))
		return nil
	}

	plmnId, err := util.PlmnIdToNgap(models.PlmnId{
		Mcc: config.Gnb.PlmnId.Mcc,
		Mnc: config.Gnb.PlmnId.Mnc,
	})
	if err != nil {
		logger.Error("CONFIG", fmt.Sprintf("Error converting plmnId to ngap: %v", err))
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
		logger.Error("CONFIG", fmt.Sprintf("Error converting tai to ngap: %v", err))
		return nil
	}

	sstInt, err := strconv.Atoi(config.Gnb.Snssai.Sst)
	if err != nil {
		logger.Error("CONFIG", fmt.Sprintf("Error converting sst to int: %v", err))
		return nil
	}
	snssai, err := util.SNssaiToNgap(models.Snssai{
		Sst: int32(sstInt),
		Sd:  config.Gnb.Snssai.Sd,
	})
	if err != nil {
		logger.Error("CONFIG", fmt.Sprintf("Error converting snssai to ngap: %v", err))
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

		logger: logger,
	}
}

func (g *Gnb) Start(ctx context.Context) error {
	g.logger.Info("GNB", "Starting GNB")
	if err := g.connectToAmf(); err != nil {
		g.logger.Error("SCTP", err.Error())
		return err
	}

	if err := g.setupN2(); err != nil {
		g.logger.Error("NGAP", fmt.Sprintf("Error setting up N2: %v", err))
		return err
	}

	if err := g.startRanListener(); err != nil {
		g.logger.Error("RAN", fmt.Sprintf("Error starting RAN listener: %v", err))
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
					g.logger.Error("RAN", fmt.Sprintf("Error accepting RAN connection: %v", err))
					continue
				}
				g.logger.Info("RAN", fmt.Sprintf("New UE connection accepted from: %v", conn.RemoteAddr()))
				g.activeConns.Store(conn, struct{}{})
				go g.handleRanConnection(ctx, conn)
			}
		}
	}()

	g.logger.Info("GNB", "GNB started")
	return nil
}

func (g *Gnb) Stop() {
	g.logger.Info("GNB", "Stopping GNB")
	if err := (*g.ranListener).Close(); err != nil {
		g.logger.Error("RAN", fmt.Sprintf("Error stopping GNB: %v", err))
		return
	}
	g.logger.Debug("RAN", fmt.Sprintf("RAN listener stopped at %s:%d", g.ranIp, g.ranPort))

	var wg sync.WaitGroup
	g.activeConns.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			if conn, ok := key.(net.Conn); ok {
				if err := conn.Close(); err != nil {
					g.logger.Error("RAN", fmt.Sprintf("Error closing UE connection: %v", err))
				}
			}
			g.logger.Debug("RAN", fmt.Sprintf("Closed UE connection from: %v", conn.RemoteAddr()))
		}(key.(net.Conn))
		return true
	})
	wg.Wait()

	if err := g.n2Conn.Close(); err != nil {
		g.logger.Error("SCTP", fmt.Sprintf("Error stopping GNB: %v", err))
		return
	}
	g.logger.Debug("SCTP", fmt.Sprintf("N2 connection closed at %s:%d", g.gnbN2Ip, g.gnbN2Port))
	g.logger.Info("GNB", "GNB stopped")
}

func (g *Gnb) connectToAmf() error {
	g.logger.Info("GNB", "Connecting to AMF")

	amfAddr, gnbAddr, err := getAmfAndGnbSctpN2Addr(g.amfN2Ip, g.gnbN2Ip, g.amfN2Port, g.gnbN2Port)
	if err != nil {
		return err
	}
	g.logger.Debug("SCTP", fmt.Sprintf("AMF N2 Address: %v", amfAddr.String()))
	g.logger.Debug("SCTP", fmt.Sprintf("GNB N2 Address: %v", gnbAddr.String()))

	conn, err := sctp.DialSCTP("sctp", gnbAddr, amfAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("Error connecting to AMF: %v", err))
	}

	info, err := conn.GetDefaultSentParam()
	if err != nil {
		return err
	}
	info.PPID = g.ngapPpid
	if err := conn.SetDefaultSentParam(info); err != nil {
		return errors.New(fmt.Sprintf("Error setting default sent param: %v", err))
	}

	g.n2Conn = conn

	g.logger.Info("GNB", fmt.Sprintf("Connected to AMF: %v", amfAddr.String()))
	return nil
}

func (g *Gnb) setupN2() error {
	g.logger.Info("GNB", "Setting up N2")

	request, err := getNgapSetupRequest(g.gnbId, g.gnbName, g.plmnId, g.tai, g.snssai)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting NGAP setup request: %v", err))
	}

	_, err = g.n2Conn.Write(request)
	if err != nil {
		return errors.New(fmt.Sprintf("Error sending NGAP setup request: %v", err))
	}

	response := make([]byte, 2048)
	responseLen, err := g.n2Conn.Read(response)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading NGAP setup response: %v", err))
	}

	responsePdu, err := ngap.Decoder(response[:responseLen])
	if err != nil {
		return errors.New(fmt.Sprintf("Error decoding NGAP setup response: %v", err))
	}

	if (responsePdu.Present != ngapType.NGAPPDUPresentSuccessfulOutcome) || (responsePdu.SuccessfulOutcome.ProcedureCode.Value != ngapType.ProcedureCodeNGSetup) {
		return errors.New(fmt.Sprintf("Error NGAP setup response: %+v", responsePdu))
	}

	g.logger.Info("NGAP", "============= gNB Info =============")

	gnbId := util.BytesToHexString(g.gnbId)
	g.logger.Info("NGAP", fmt.Sprintf("gNB ID: %v, name: %s", gnbId, g.gnbName))

	plmnId := ngapConvert.PlmnIdToModels(g.plmnId)
	g.logger.Info("NGAP", fmt.Sprintf("PLMN ID: %v", plmnId))

	tai := ngapConvert.TaiToModels(g.tai)
	g.logger.Info("NGAP", fmt.Sprintf("TAC: %v, broadcast PLMN ID: %v", tai.Tac, tai.PlmnId))

	snssai := ngapConvert.SNssaiToModels(g.snssai)
	g.logger.Info("NGAP", fmt.Sprintf("SST: %v, SD: %v", snssai.Sst, snssai.Sd))

	g.logger.Info("NGAP", "====================================")

	g.logger.Info("GNB", "N2 setup complete")
	return nil
}

func (g *Gnb) setupN1(n1Conn net.Conn) error {
	g.logger.Info("GNB", "Setting up N1")

	// ue initialization
	mobileIdentity5GS, err := g.processUeInitialization(n1Conn)
	if err != nil {
		return err
	}

	g.logger.Info("GNB", fmt.Sprintf("UE %s N1 setup complete", mobileIdentity5GS.GetSUCI()))
	return nil
}

func (g *Gnb) startRanListener() error {
	g.logger.Info("GNB", "Starting RAN listener")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.ranIp, g.ranPort))
	if err != nil {
		return err
	}
	g.ranListener = &listener

	g.logger.Info("RAN", "============= RAN Info =============")
	g.logger.Info("RAN", fmt.Sprintf("RAN access address: %s:%d", g.ranIp, g.ranPort))
	g.logger.Info("RAN", "====================================")

	g.logger.Info("GNB", "RAN listener started")
	return nil
}

func (g *Gnb) handleRanConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			g.logger.Error("RAN", fmt.Sprintf("Error closing UE connection: %v", err))
		}
		g.logger.Info("RAN", fmt.Sprintf("Closed UE connection from: %v", conn.RemoteAddr()))
		g.activeConns.Delete(conn)
	}()

	if err := g.setupN1(conn); err != nil {
		g.logger.Error("RAN", err.Error())
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
					g.logger.Debug("RAN", fmt.Sprintf("UE connection closed by client: %v", conn.RemoteAddr()))
					return
				}
				g.logger.Error("RAN", fmt.Sprintf("Error reading from UE connection: %v", err))
				return
			}
		}
	}
}

func (g *Gnb) processUeInitialization(n1Conn net.Conn) (nasType.MobileIdentity5GS, error) {
	g.logger.Info("RAN", "Processing UE initialization")

	var mobileIdentity5GS nasType.MobileIdentity5GS

	// receive ue registration request and send initial ue message to AMF
	ueRegistrationRequest := make([]byte, 1024)
	if _, err := n1Conn.Read(ueRegistrationRequest); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive ue registration request from UE: %v", err))
	}
	nasMessage := nas.NewMessage()
	if err := nasMessage.GmmMessageDecode(&ueRegistrationRequest); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error decode ue registration request from UE: %v", err))
	}
	mobileIdentity5GS = nasMessage.GmmMessage.RegistrationRequest.MobileIdentity5GS
	g.logger.Debug("RAN", fmt.Sprintf("Receive UE %s registration request from UE", mobileIdentity5GS.GetSUCI()))

	ueInitialMessage, err := getInitialUeMessage(1, ueRegistrationRequest, g.plmnId, g.tai)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get initial ue message: %v", err))
	}

	if _, err := g.n2Conn.Write(ueInitialMessage); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send initial ue message to AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send initial UE message to AMF"))

	// receive nas authentication request from AMF and send to UE
	nasAuthenticationRequest := make([]byte, 1024)
	n, err := g.n2Conn.Read(nasAuthenticationRequest)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive initial ue response from AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NAS Authentication Request from AMF"))

	if _, err := n1Conn.Write(nasAuthenticationRequest[:n]); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send nas authentication request to UE: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NAS Authentication Request to UE"))

	// receive nas authentication response from UE and send to AMF
	nasAuthenticationResponse := make([]byte, 1024)
	n, err = n1Conn.Read(nasAuthenticationResponse)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas authentication response from UE: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NAS Authentication Response from UE"))

	uplinkNasTransport, err := getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasAuthenticationResponse[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	if _, err := g.n2Conn.Write(uplinkNasTransport); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send uplink nas transport to AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NAS Authentication Response to AMF"))

	// receive nas security mode command message from AMF and send to UE
	nasSecurityModeCommand := make([]byte, 1024)
	n, err = g.n2Conn.Read(nasSecurityModeCommand)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas security mode command from AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NAS Security Mode Command from AMF"))

	if _, err := n1Conn.Write(nasSecurityModeCommand[:n]); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send nas security mode command to UE: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NAS Security Mode Command to UE"))

	// receive nas security mode complete message from UE and send to AMF
	nasSecurityModeComplete := make([]byte, 1024)
	n, err = n1Conn.Read(nasSecurityModeComplete)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas security mode complete from UE: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NAS Security Mode Complete from UE"))

	uplinkNasTransport, err = getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasSecurityModeComplete[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	if _, err := g.n2Conn.Write(uplinkNasTransport); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send uplink nas transport to AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NAS Security Mode Complete to AMF"))

	// receive ngap initial context setup request from AMF
	ngapInitialContextSetupRequestRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ngapInitialContextSetupRequestRaw)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive ngap initial context setup request from AMF: %v", err))
	}

	ngapInitialContextSetupRequest, err := ngap.Decoder(ngapInitialContextSetupRequestRaw[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error decode ngap initial context setup request from AMF: %v", err))
	}
	if ngapInitialContextSetupRequest.Present != ngapType.NGAPPDUPresentInitiatingMessage || ngapInitialContextSetupRequest.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeInitialContextSetup {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error ngap initial context setup request: no initial context setup request"))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NGAP Initial Context Setup Request from AMF"))

	// send ngap initial context setup response to AMF
	ngapInitialContextSetupResponse, err := getNgapInitialContextSetupResponse(1, 1)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get ngap initial context setup response: %v", err))
	}
	if _, err := g.n2Conn.Write(ngapInitialContextSetupResponse); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send ngap initial context setup response to AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NGAP Initial Context Setup Response to AMF"))

	// receive nas registration complete message from UE and send to AMF
	nasRegistrationComplete := make([]byte, 1024)
	n, err = n1Conn.Read(nasRegistrationComplete)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive nas registration complete from UE: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive NAS Registration Complete from UE"))

	uplinkNasTransport, err = getUplinkNasTransport(1, 1, g.plmnId, g.tai, nasRegistrationComplete[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error get uplink nas transport: %v", err))
	}
	if _, err := g.n2Conn.Write(uplinkNasTransport); err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error send uplink nas transport to AMF: %v", err))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Send NAS Registration Complete to AMF"))

	// receive ue configuration update command message from AMF
	ueConfigurationUpdateCommandRaw := make([]byte, 1024)
	n, err = g.n2Conn.Read(ueConfigurationUpdateCommandRaw)
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error receive ue configuration update command from AMF: %v", err))
	}
	ueConfigurationUpdateCommand, err := ngap.Decoder(ueConfigurationUpdateCommandRaw[:n])
	if err != nil {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error decode ue configuration update command from AMF: %v", err))
	}
	if ueConfigurationUpdateCommand.Present != ngapType.NGAPPDUPresentInitiatingMessage || ueConfigurationUpdateCommand.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		return mobileIdentity5GS, errors.New(fmt.Sprintf("Error ue configuration update command: no ue configuration update command"))
	}
	g.logger.Debug("RAN", fmt.Sprintf("Receive UE Configuration Update Command from AMF"))

	g.logger.Info("RAN", fmt.Sprintf("UE %s initialized", mobileIdentity5GS.GetSUCI()))
	return mobileIdentity5GS, nil
}
