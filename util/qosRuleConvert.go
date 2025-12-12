package util

import (
	"fmt"

	"github.com/free-ran-ue/free-ran-ue/logger"
	"github.com/free5gc/nas/nasType"
)

func GetQosRule(ruleBytes []byte, logger *logger.UeLogger) []string {
	var rules nasType.QoSRules
	if err := rules.UnmarshalBinary(ruleBytes); err != nil {
		logger.PduLog.Warnf("unmarshal qos rules failed: %+v", err)
		return nil
	}

	qosRules := make([]string, 0)

	for _, r := range rules {
		for _, p := range r.PacketFilterList {
			for _, c := range p.Components {
				switch c.Type() {
				case nasType.PacketFilterComponentTypeMatchAll:
				case nasType.PacketFilterComponentTypeIPv4RemoteAddress:
					value := c.(*nasType.PacketFilterIPv4RemoteAddress)
					ip := value.Address.String()
					maskLen, _ := value.Mask.Size()
					qosRules = append(qosRules, fmt.Sprintf("%s/%d", ip, maskLen))
				default:
					logger.PduLog.Warnf("unsupported qos rule component type: %d", c.Type())
				}
			}
		}
	}

	return qosRules
}
