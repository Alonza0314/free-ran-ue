package model

type TaiIE struct {
	Tac             string   `yaml:"tac"`
	BroadcastPlmnId PlmnIdIE `yaml:"broadcastPlmnId"`
}

type PlmnIdIE struct {
	Mcc string `yaml:"mcc"`
	Mnc string `yaml:"mnc"`
}
