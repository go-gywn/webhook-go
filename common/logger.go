package common

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

// Logger logger
type Logger struct {
	infoFunc  func(params ...interface{})
	errorFunc func(params ...interface{})
	debugFunc func(params ...interface{})
}

// NewLogger NewLogger
func NewLogger(name string, level ...string) Logger {
	if len(level) == 0 {
		return NewLogger(name, getDefaultLogLevel())
	}
	logger := Logger{}
	switch strings.ToUpper(level[0]) {
	case "DEBUG":
		logger.errorFunc = getLoggerError(name)
		logger.infoFunc = getLoggerInfo(name)
		logger.debugFunc = getLoggerDebug(name)
		gin.SetMode(gin.DebugMode)
	case "TEST":
		logger.errorFunc = getLoggerError(name)
		logger.infoFunc = getLoggerInfo(name)
		logger.debugFunc = getLoggerNone(name)
		gin.SetMode(gin.TestMode)
	case "RELEASE":
		logger.errorFunc = getLoggerError(name)
		logger.infoFunc = getLoggerNone(name)
		logger.debugFunc = getLoggerNone(name)
		gin.SetMode(gin.ReleaseMode)
	default:
		return NewLogger(name, "DEBUG")
	}
	return logger
}

func getDefaultLogLevel() string {
	return Cfg.LogLevel
}

// Info print info log
func (l *Logger) Info(params ...interface{}) {
	l.infoFunc(params...)
}

// Error print error log
func (l *Logger) Error(params ...interface{}) {
	l.errorFunc(params...)
}

// Debug print debug log
func (l *Logger) Debug(params ...interface{}) {
	l.debugFunc(params...)
}

func getLoggerInfo(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", green, "INFO", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func getLoggerError(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", red, "ERROR", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func getLoggerDebug(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", white, "DEBUG", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func getLoggerNone(name string) func(params ...interface{}) {
	return func(params ...interface{}) {
	}
}
