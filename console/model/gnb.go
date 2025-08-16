package model

type ConsoleGnbInfoRequest struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

type ConsoleGnbInfoResponse struct {
	Message string  `json:"message"`
	GnbInfo GnbInfo `json:"gnbInfo"`
}

type GnbInfo struct {
	GnbId   string `json:"gnbId"`
	GnbName string `json:"gnbName"`

	PlmnId string `json:"plmnId"`

	Snssai SnssaiIE `json:"snssai"`

	RanUeList []RanUeInfo `json:"ranUeList"`
	XnUeList  []XnUeInfo  `json:"xnUeList"`
}

type SnssaiIE struct {
	Sst string `json:"sst"`
	Sd  string `json:"sd"`
}

type RanUeInfo struct {
	Imsi          string `json:"imsi"`
	NrdcIndicator bool   `json:"nrdcIndicator"`
}

type XnUeInfo struct {
	Imsi string `json:"imsi"`
}

type ConsoleGnbUeNrdcModifyRequest struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
	Imsi string `json:"imsi"`
}

type ConsoleGnbUeNrdcModifyResponse struct {
	Message string `json:"message"`
}
