package logger

import (
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

type ConsoleLogger struct {
	*loggergo.Logger

	CfgLog     loggergoModel.LoggerInterface
	ConsoleLog loggergoModel.LoggerInterface
	LoginLog   loggergoModel.LoggerInterface
	LogoutLog  loggergoModel.LoggerInterface
	AuthLog    loggergoModel.LoggerInterface
	GnbLog     loggergoModel.LoggerInterface
}

func NewConsoleLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) ConsoleLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return ConsoleLogger{
		Logger: logger,

		CfgLog:     logger.WithTags(CSL_TAG, CONFIG_TAG),
		ConsoleLog: logger.WithTags(CSL_TAG, CONSOLE_TAG),
		LoginLog:   logger.WithTags(CSL_TAG, LOGIN_TAG),
		LogoutLog:  logger.WithTags(CSL_TAG, LOGOUT_TAG),
		AuthLog:    logger.WithTags(CSL_TAG, AUTH_TAG),
		GnbLog:     logger.WithTags(CSL_TAG, GNB_TAG),
	}
}
