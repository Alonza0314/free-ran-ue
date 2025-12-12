package logger

import (
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/free-ran-ue/constant"
)

type UeLogger struct {
	*loggergo.Logger

	CfgLog loggergoModel.LoggerInterface
	UeLog  loggergoModel.LoggerInterface
	RanLog loggergoModel.LoggerInterface
	NasLog loggergoModel.LoggerInterface
	PduLog loggergoModel.LoggerInterface
	TunLog loggergoModel.LoggerInterface
}

func NewUeLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) UeLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return UeLogger{
		Logger: logger,

		CfgLog: logger.WithTags(constant.UE_TAG, constant.CONFIG_TAG),
		UeLog:  logger.WithTags(constant.UE_TAG, constant.UE_TAG),
		RanLog: logger.WithTags(constant.UE_TAG, constant.RAN_TAG),
		NasLog: logger.WithTags(constant.UE_TAG, constant.NAS_TAG),
		PduLog: logger.WithTags(constant.UE_TAG, constant.PDU_TAG),
		TunLog: logger.WithTags(constant.UE_TAG, constant.TUN_TAG),
	}
}
