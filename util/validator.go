package util

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"

	"github.com/Alonza0314/free-ran-ue/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free5gc/openapi/models"
)

/*
As before using validator, we have loaded the config from yaml file.
In the loaded function, it is ensured that the Data Type is correct.
All the validator functions need to do is to ensure the data value is valid.
*/

func ValidateLoggerIe(loggerIe *model.LoggerIE) error {
	switch loggergoUtil.LogLevelString(loggerIe.Level) {
	case loggergoUtil.LEVEL_STRING_ERROR:
		return nil
	case loggergoUtil.LEVEL_STRING_WARN:
		return nil
	case loggergoUtil.LEVEL_STRING_INFO:
		return nil
	case loggergoUtil.LEVEL_STRING_DEBUG:
		return nil
	case loggergoUtil.LEVEL_STRING_TRACE:
		return nil
	case loggergoUtil.LEVEL_STRING_TEST:
		return nil
	default:
		return fmt.Errorf("invalid logger level: %s", loggerIe.Level)
	}
}

func ValidateIp(ip string) error {
	ipAddress := net.ParseIP(ip)
	if ipAddress == nil {
		return fmt.Errorf("invalid ip address: %s", ip)
	}
	return nil
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port range: %d, range should be 1-65535", port)
	}
	return nil
}

func ValidateIntStringWithLength(intString string, length int) error {
	if _, err := strconv.Atoi(intString); err != nil {
		return fmt.Errorf("invalid int string: %s", intString)
	}
	if len(intString) != length {
		return fmt.Errorf("invalid int string: %s, length should be %d", intString, length)
	}
	return nil
}

func ValidatePlmnId(plmnId *model.PlmnIdIE) error {
	if len(plmnId.Mcc) != 3 {
		return fmt.Errorf("invalid mcc: %s, mcc should be 3 digits", plmnId.Mcc)
	}
	if err := ValidateIntStringWithLength(plmnId.Mcc, 3); err != nil {
		return err
	}
	if len(plmnId.Mnc) != 2 {
		return fmt.Errorf("invalid mnc: %s, mnc should be 2 digits", plmnId.Mnc)
	}
	if err := ValidateIntStringWithLength(plmnId.Mnc, 2); err != nil {
		return err
	}
	return nil
}

func ValidateMsin(msin string) error {
	if len(msin) != 10 {
		return fmt.Errorf("invalid msin: %s, msin should be 10 digits", msin)
	}
	if err := ValidateIntStringWithLength(msin, 10); err != nil {
		return err
	}
	return nil
}

func ValidateAccessType(accessType models.AccessType) error {
	switch accessType {
	case models.AccessType__3_GPP_ACCESS:
		return nil
	case models.AccessType_NON_3_GPP_ACCESS:
		return fmt.Errorf("unsupported access type: %s", accessType)
	default:
		return fmt.Errorf("invalid access type: %s", accessType)
	}
}

func ValidateHexString(hexString string) error {
	if _, err := hex.DecodeString(hexString); err != nil {
		return fmt.Errorf("invalid hex string: %s", hexString)
	}
	return nil
}

func ValidateAuthenticationSubscription(authenticationSubscription *model.AuthenticationSubscriptionIE) error {
	if err := ValidateHexString(authenticationSubscription.EncPermanentKey); err != nil {
		return fmt.Errorf("invalid enc permanent key, %s", err.Error())
	}
	if err := ValidateHexString(authenticationSubscription.EncOpcKey); err != nil {
		return fmt.Errorf("invalid enc opc key, %s", err.Error())
	}
	if err := ValidateIntStringWithLength(authenticationSubscription.AuthenticationManagementField, 4); err != nil {
		return fmt.Errorf("invalid authentication management field, %s", err.Error())
	}
	if err := ValidateIntStringWithLength(authenticationSubscription.SequenceNumber, 12); err != nil {
		return fmt.Errorf("invalid sequence number, %s", err.Error())
	}
	return nil
}

func ValidateXorBooleanFlag(booleanFlags ...bool) error {
	trueExist := false
	for _, booleanFlag := range booleanFlags {
		if booleanFlag {
			if trueExist {
				return fmt.Errorf("exist multiple true boolean flags")
			} else {
				trueExist = true
			}
		}
	}
	if !trueExist {
		return fmt.Errorf("no true boolean flag, one true flag is required")
	}
	return nil
}

func ValidateIntegrityAlgorithm(integrityAlgorithm *model.IntegrityAlgorithmIE) error {
	return ValidateXorBooleanFlag(integrityAlgorithm.Nia0, integrityAlgorithm.Nia1, integrityAlgorithm.Nia2, integrityAlgorithm.Nia3)
}

func ValidateCipheringAlgorithm(cipheringAlgorithm *model.CipheringAlgorithmIE) error {
	return ValidateXorBooleanFlag(cipheringAlgorithm.Nea0, cipheringAlgorithm.Nea1, cipheringAlgorithm.Nea2, cipheringAlgorithm.Nea3)
}

func ValidatePduSession(pduSession *model.PduSessionIE) error {
	if err := ValidateIntStringWithLength(pduSession.Snssai.Sst, 1); err != nil {
		return fmt.Errorf("invalid pdu session sst, %s", err.Error())
	}
	if err := ValidateHexString(pduSession.Snssai.Sd); err != nil {
		return fmt.Errorf("invalid pdu session sd, %s", err.Error())
	}
	return nil
}

func ValidateNrdc(nrdc *model.NrdcIE) error {
	if !nrdc.Enable {
		return nil
	}
	if err := ValidateIp(nrdc.DcRanDataPlane.Ip); err != nil {
		return fmt.Errorf("invalid nrdc dc ran data plane ip, %s", err.Error())
	}
	if err := ValidatePort(nrdc.DcRanDataPlane.Port); err != nil {
		return fmt.Errorf("invalid nrdc dc ran data plane port, %s", err.Error())
	}
	return nil
}

func ValidateUeIe(ueIe *model.UeIE) error {
	if err := ValidateIp(ueIe.RanControlPlaneIp); err != nil {
		return fmt.Errorf("invalid ue ran control plane ip, %s", err.Error())
	}
	if err := ValidateIp(ueIe.RanDataPlaneIp); err != nil {
		return fmt.Errorf("invalid ue ran data plane ip, %s", err.Error())
	}

	if err := ValidatePort(ueIe.RanControlPlanePort); err != nil {
		return fmt.Errorf("invalid ue ran control plane port, %s", err.Error())
	}
	if err := ValidatePort(ueIe.RanDataPlanePort); err != nil {
		return fmt.Errorf("invalid ue ran data plane port, %s", err.Error())
	}

	if err := ValidatePlmnId(&ueIe.PlmnId); err != nil {
		return fmt.Errorf("invalid ue plmn id, %s", err.Error())
	}
	if err := ValidateMsin(ueIe.Msin); err != nil {
		return fmt.Errorf("invalid ue msin, %s", err.Error())
	}

	if err := ValidateAccessType(ueIe.AccessType); err != nil {
		return fmt.Errorf("invalid ue access type, %s", err.Error())
	}
	if err := ValidateAuthenticationSubscription(&ueIe.AuthenticationSubscription); err != nil {
		return fmt.Errorf("invalid ue authentication subscription, %s", err.Error())
	}

	if err := ValidateCipheringAlgorithm(&ueIe.CipheringAlgorithm); err != nil {
		return fmt.Errorf("invalid ue ciphering algorithm, %s", err.Error())
	}
	if err := ValidateIntegrityAlgorithm(&ueIe.IntegrityAlgorithm); err != nil {
		return fmt.Errorf("invalid ue integrity algorithm, %s", err.Error())
	}

	if err := ValidatePduSession(&ueIe.PduSession); err != nil {
		return fmt.Errorf("invalid ue pdu session, %s", err.Error())
	}

	if err := ValidateNrdc(&ueIe.Nrdc); err != nil {
		return fmt.Errorf("invalid ue nrdc, %s", err.Error())
	}
	return nil
}

func ValidateUe(ue *model.UeConfig) error {
	if err := ValidateUeIe(&ue.Ue); err != nil {
		return err
	}
	if err := ValidateLoggerIe(&ue.Logger); err != nil {
		return err
	}
	return nil
}
