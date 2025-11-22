package constant

import "net/http"

type UeType string

// for cmd
const (
	BASIC_UE_NUM            = 1
	BASIC_UE_MAX_CONCURRENT = 10
)

// for RAN
const (
	NGAP_PPID uint32 = 0x3c000000

	UE_TYPE_RAN UeType = "ran"
	UE_TYPE_XN  UeType = "xn"
)

// for UE
const (
	PDU_SESSION_ID = 4
)

// between RAN and UE
const (
	UE_DATA_PLANE_INITIAL_PACKET = "initial packet"
	UE_TUNNEL_UPDATE             = "tunnel update"
	UE_IMSI_PREFIX               = "imsi-"
)

// for logger
const (
	CONFIG_TAG = "CONFIG"

	GNB_TAG = "GNB"

	RAN_TAG = "RAN"
	UE_TAG  = "UE"
	CSL_TAG = "CSL"

	SCTP_TAG = "SCTP"
	UDP_TAG  = "UDP"
	NGAP_TAG = "NGAP"
	NAS_TAG  = "NAS"
	PDU_TAG  = "PDU"
	GTP_TAG  = "GTP"
	TUN_TAG  = "TUN"

	XN_TAG = "XN"

	CONSOLE_TAG = "CONSOLE"
	LOGIN_TAG   = "LOGIN"
	LOGOUT_TAG  = "LOGOUT"
	AUTH_TAG    = "AUTH"

	API_TAG = "API"
)

// for gtp
const (
	IS_NEXT_EXTENSION_HEADER = 0x04
	IS_SEQUENCE_NUMBER       = 0x02
	IS_N_PDU_NUMBER          = 0x01

	NEXT_EXTENSION_HEADER_TYPE_NO_MORE_EXTENSION_HEADERS = 0x00

	NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER        = 0x85
	NEXT_EXTENSION_HEADER_TYPE_PDU_SESSION_CONTAINER_LENGTH = 2
)

// API_PREFIX defines API path prefixes for gin
type API_PREFIX string

// for gin
const (
	API_PREFIX_GNB     API_PREFIX = "/api/gnb"
	API_PREFIX_UE      API_PREFIX = "/api/ue"
	API_PREFIX_CONSOLE API_PREFIX = "/api/console"
)

// for gNB
const (
	API_GNB_INFO        = "/info"
	API_GNB_INFO_METHOD = http.MethodGet

	API_GNB_UE_NRDC        = "/ue/nrdc"
	API_GNB_UE_NRDC_METHOD = http.MethodPost
)

// for console
const (
	APPLICATION_JSON = "application/json"

	API_REQUEST_GNB_INFO        = API_PREFIX_GNB + API_GNB_INFO
	API_REQUEST_GNB_INFO_METHOD = API_GNB_INFO_METHOD

	API_REQUEST_GNB_UE_NRDC        = API_PREFIX_GNB + API_GNB_UE_NRDC
	API_REQUEST_GNB_UE_NRDC_METHOD = API_GNB_UE_NRDC_METHOD
)
