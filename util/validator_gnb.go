package util

import (
	"fmt"

	"github.com/Alonza0314/free-ran-ue/model"
)

/*
As before using validator, we have loaded the config from yaml file.
In the loaded function, it is ensured that the Data Type is correct.
All the validator functions need to do is to ensure the data value is valid.
*/
func ValidateSnssaiIe(snssai *model.SnssaiIE) error {
	if err := ValidateIntStringWithLength(snssai.Sst, 1); err != nil {
		return fmt.Errorf("invalid sst, %s", err.Error())
	}

	if err := ValidateHexString(snssai.Sd); err != nil {
		return fmt.Errorf("invalid sd, %s", err.Error())
	}
	return nil
}

func ValidateTaiIe(taiIe *model.TaiIE) error {
	if err := ValidateHexString(taiIe.Tac); err != nil {
		return fmt.Errorf("invalid tac: %s", err.Error())
	}

	if err := ValidatePlmnId(&taiIe.BroadcastPlmnId); err != nil {
		return fmt.Errorf("invalid broadcastPlmnId: %s", err.Error())
	}
	return nil
}

func ValidateApiIe(apiIe *model.ApiIE) error {
	if err := ValidateIp(apiIe.Ip); err != nil {
		return fmt.Errorf("invalid ip: %s", err.Error())
	}

	if err := ValidatePort(apiIe.Port); err != nil {
		return fmt.Errorf("invalid port: %s", err.Error())
	}
	return nil
}

func ValidateXnInterfaceIe(xnIe *model.XnInterfaceIE) error {
	if !xnIe.Enable {
		return nil
	}

	if err := ValidateIp(xnIe.XnListenIp); err != nil {
		return fmt.Errorf("invalid xnListenIp: %s", err.Error())
	}
	if err := ValidatePort(xnIe.XnListenPort); err != nil {
		return fmt.Errorf("invalid xnListenPort: %s", err.Error())
	}
	if err := ValidateIp(xnIe.XnDialIp); err != nil {
		return fmt.Errorf("invalid xnDialIp: %s", err.Error())
	}
	if err := ValidatePort(xnIe.XnDialPort); err != nil {
		return fmt.Errorf("invalid xnDialPort: %s", err.Error())
	}

	return nil
}

func ValidateGnbIe(gnbIe *model.GnbIE) error {
	//validate ip for gnb
	ips := map[string]string{
		"amfN2Ip":           gnbIe.AmfN2Ip,
		"ranN2Ip":           gnbIe.RanN2Ip,
		"upfN3Ip":           gnbIe.UpfN3Ip,
		"ranN3Ip":           gnbIe.RanN3Ip,
		"ranControlPlaneIp": gnbIe.RanControlPlaneIp,
		"ranDataPlaneIp":    gnbIe.RanDataPlaneIp,
	}
	for name, ip := range ips {
		if err := ValidateIp(ip); err != nil {
			return fmt.Errorf("invalid gnb %s, %s", name, err.Error())
		}
	}

	// validate port for gnb
	ports := map[string]int{
		"amfN2Port":           gnbIe.AmfN2Port,
		"ranN2Port":           gnbIe.RanN2Port,
		"upfN3Port":           gnbIe.UpfN3Port,
		"ranN3Port":           gnbIe.RanN3Port,
		"ranControlPlanePort": gnbIe.RanControlPlanePort,
		"ranDataPlanePort":    gnbIe.RanDataPlanePort,
	}
	for name, port := range ports {
		if err := ValidatePort(port); err != nil {
			return fmt.Errorf("invalid gnb %s, %s", name, err.Error())
		}
	}

	if err := ValidateHexString(gnbIe.GnbId); err != nil {
		return fmt.Errorf("invalid gnb gnbId: %s", err.Error())
	}

	if err := ValidatePlmnId(&gnbIe.PlmnId); err != nil {
		return fmt.Errorf("invalid gnb plmn id, %s", err.Error())
	}

	if err := ValidateTaiIe(&gnbIe.Tai); err != nil {
		return fmt.Errorf("invalid gnb tai: %s", err.Error())
	}

	if err := ValidateSnssaiIe(&gnbIe.Snssai); err != nil {
		return fmt.Errorf("invalid gnb snssai: %s", err.Error())
	}

	if err := ValidateApiIe(&gnbIe.Api); err != nil {
		return fmt.Errorf("invalid gnb api: %s", err.Error())
	}

	if err := ValidateXnInterfaceIe(&gnbIe.XnInterface); err != nil {
		return fmt.Errorf("invalid gnb xnInterface: %s", err.Error())
	}

	return nil
}

func ValidateGnb(gnb *model.GnbConfig) error {
	if err := ValidateGnbIe(&gnb.Gnb); err != nil {
		return err
	}
	if err := ValidateLoggerIe(&gnb.Logger); err != nil {
		return err
	}
	return nil
}
