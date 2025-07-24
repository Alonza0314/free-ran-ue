package model

import "github.com/free5gc/openapi/models"

type UeConfig struct {
	Ue     UeIE     `yaml:"ue"`
	Logger LoggerIE `yaml:"logger"`
}

type UeIE struct {
	RanControlPlaneIp string `yaml:"ranControlPlaneIp"`
	RanDataPlaneIp    string `yaml:"ranDataPlaneIp"`

	RanControlPlanePort int `yaml:"ranControlPlanePort"`
	RanDataPlanePort    int `yaml:"ranDataPlanePort"`

	PlmnId PlmnIdIE `yaml:"plmnId"`
	Msin   string   `yaml:"msin"`

	AccessType                 models.AccessType            `yaml:"accessType"`
	AuthenticationSubscription AuthenticationSubscriptionIE `yaml:"authenticationSubscription"`

	CipheringAlgorithm CipheringAlgorithmIE `yaml:"cipheringAlgorithm"`
	IntegrityAlgorithm IntegrityAlgorithmIE `yaml:"integrityAlgorithm"`

	PduSession PduSessionIE `yaml:"pduSession"`

	Nrdc NrdcIE `yaml:"nrdc"`

	UeTunnelDevice string `yaml:"ueTunnelDevice"`
}

type AuthenticationSubscriptionIE struct {
	EncPermanentKey               string `yaml:"encPermanentKey"`
	EncOpcKey                     string `yaml:"encOpcKey"`
	AuthenticationManagementField string `yaml:"authenticationManagementField"`
	SequenceNumber                string `yaml:"sequenceNumber"`
}

type IntegrityAlgorithmIE struct {
	Nia0 bool `yaml:"nia0"`
	Nia1 bool `yaml:"nia1"`
	Nia2 bool `yaml:"nia2"`
	Nia3 bool `yaml:"nia3"`
}

type CipheringAlgorithmIE struct {
	Nea0 bool `yaml:"nea0"`
	Nea1 bool `yaml:"nea1"`
	Nea2 bool `yaml:"nea2"`
	Nea3 bool `yaml:"nea3"`
}

type PduSessionIE struct {
	PduSessionId uint8    `yaml:"pduSessionId"`
	Dnn          string   `yaml:"dnn"`
	Snssai       SnssaiIE `yaml:"snssai"`
}

type NrdcIE struct {
	Enable         bool          `yaml:"enable"`
	DcRanDataPlane DcDataPlaneIE `yaml:"dcRanDataPlane"`
	QosFlow        []string      `yaml:"qosFlow"`
}

type DcDataPlaneIE struct {
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
}
