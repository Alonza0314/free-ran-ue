package logger

import (
	"github.com/Alonza0314/free-ran-ue/constant"
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

type GnbLogger struct {
	*loggergo.Logger

	CfgLog  loggergoModel.LoggerInterface
	RanLog  loggergoModel.LoggerInterface
	SctpLog loggergoModel.LoggerInterface
	NgapLog loggergoModel.LoggerInterface
	NasLog  loggergoModel.LoggerInterface
	GtpLog  loggergoModel.LoggerInterface
	XnLog   loggergoModel.LoggerInterface
	ApiLog  loggergoModel.LoggerInterface
}

func NewGnbLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) GnbLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return GnbLogger{
		Logger: logger,

		CfgLog:  logger.WithTags(constant.GNB_TAG, constant.CONFIG_TAG),
		RanLog:  logger.WithTags(constant.GNB_TAG, constant.RAN_TAG),
		SctpLog: logger.WithTags(constant.GNB_TAG, constant.SCTP_TAG),
		NgapLog: logger.WithTags(constant.GNB_TAG, constant.NGAP_TAG),
		NasLog:  logger.WithTags(constant.GNB_TAG, constant.NAS_TAG),
		GtpLog:  logger.WithTags(constant.GNB_TAG, constant.GTP_TAG),
		XnLog:   logger.WithTags(constant.GNB_TAG, constant.XN_TAG),
		ApiLog:  logger.WithTags(constant.GNB_TAG, constant.API_TAG),
	}
}
