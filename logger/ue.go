package logger

import (
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

type UeLogger struct {
	*loggergo.Logger

	CfgLog loggergoModel.LoggerInterface
	UeLog  loggergoModel.LoggerInterface
	TcpLog loggergoModel.LoggerInterface
	NasLog loggergoModel.LoggerInterface
}

func NewUeLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) UeLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return UeLogger{
		Logger: logger,

		CfgLog: logger.WithTags(UE_TAG, CONFIG_TAG),
		UeLog:  logger.WithTags(UE_TAG, UE_TAG),
		TcpLog: logger.WithTags(UE_TAG, TCP_TAG),
		NasLog: logger.WithTags(UE_TAG, NAS_TAG),
	}
}
