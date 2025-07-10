package model

type GnbConfig struct {
	Gnb    GnbIE    `yaml:"gnb"`
	Logger LoggerIE `yaml:"logger"`
}

type GnbIE struct {
	AmfN2Ip string `yaml:"amfN2Ip"`
	GnbN2Ip string `yaml:"gnbN2Ip"`
	RanIp   string `yaml:"ranIp"`

	AmfN2Port int `yaml:"amfN2Port"`
	GnbN2Port int `yaml:"gnbN2Port"`
	RanPort   int `yaml:"ranPort"`

	NgapPpid uint32 `yaml:"ngapPpid"`

	GnbId   string `yaml:"gnbId"`
	GnbName string `yaml:"gnbName"`

	PlmnId PlmnIdIE `yaml:"plmnId"`

	Tai    TaiIE    `yaml:"tai"`
	Snssai SnssaiIE `yaml:"snssai"`
}

type SnssaiIE struct {
	Sst string `yaml:"sst"`
	Sd  string `yaml:"sd"`
}
