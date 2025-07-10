package logger

import (
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
}

func NewGnbLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) GnbLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return GnbLogger{
		Logger: logger,

		CfgLog:  logger.WithTags(GNB_TAG, CONFIG_TAG),
		RanLog:  logger.WithTags(GNB_TAG, RAN_TAG),
		SctpLog: logger.WithTags(GNB_TAG, SCTP_TAG),
		NgapLog: logger.WithTags(GNB_TAG, NGAP_TAG),
		NasLog:  logger.WithTags(GNB_TAG, NAS_TAG),
	}
}
