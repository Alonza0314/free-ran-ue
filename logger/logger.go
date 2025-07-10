package logger

// import (
// 	"fmt"

// 	logger "github.com/Alonza0314/logger-go/v2"
// )

// type Logger struct {
// 	Level LoggerLevel
// }

// func NewLogger(level string) (*Logger, error) {
// 	var levelInt LoggerLevel

// 	switch level {
// 	case ERROR:
// 		levelInt = LOGGER_ERROR
// 	case WARN:
// 		levelInt = LOGGER_WARN
// 	case INFO:
// 		levelInt = LOGGER_INFO
// 	case DEBUG:
// 		levelInt = LOGGER_DEBUG
// 	default:
// 		return nil, fmt.Errorf("invalid logger level: %s", level)
// 	}

// 	return &Logger{
// 		Level: levelInt,
// 	}, nil
// }

// func (l *Logger) Error(tag, msg string) {
// 	if l.Level < LOGGER_ERROR {
// 		return
// 	}
// 	logger.Error(tag, msg)
// }

// func (l *Logger) Warn(tag, msg string) {
// 	if l.Level < LOGGER_WARN {
// 		return
// 	}
// 	logger.Warn(tag, msg)
// }

// func (l *Logger) Info(tag, msg string) {
// 	if l.Level < LOGGER_INFO {
// 		return
// 	}
// 	logger.Info(tag, msg)
// }

// func (l *Logger) Debug(tag, msg string) {
// 	if l.Level < LOGGER_DEBUG {
// 		return
// 	}
// 	logger.Debug(tag, msg)
// }
