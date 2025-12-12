package logger

import (
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/free-ran-ue/constant"
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

		CfgLog:     logger.WithTags(constant.CSL_TAG, constant.CONFIG_TAG),
		ConsoleLog: logger.WithTags(constant.CSL_TAG, constant.CONSOLE_TAG),
		LoginLog:   logger.WithTags(constant.CSL_TAG, constant.LOGIN_TAG),
		LogoutLog:  logger.WithTags(constant.CSL_TAG, constant.LOGOUT_TAG),
		AuthLog:    logger.WithTags(constant.CSL_TAG, constant.AUTH_TAG),
		GnbLog:     logger.WithTags(constant.CSL_TAG, constant.GNB_TAG),
	}
}
