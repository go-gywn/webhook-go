package goutil

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	Level     string
	infoFunc  func(params ...interface{})
	errorFunc func(params ...interface{})
	debugFunc func(params ...interface{})
}

// GetLogger get logger
func GetLogger(name string) Logger {
	var logger = Logger{Level: GetLogMode()}
	switch logger.Level {
	case "debug":
		logger.errorFunc = logger.getLoggerError(name)
		logger.infoFunc = logger.getLoggerInfo(name)
		logger.debugFunc = logger.getLoggerDebug(name)
	case "test":
		logger.errorFunc = logger.getLoggerError(name)
		logger.infoFunc = logger.getLoggerInfo(name)
		logger.debugFunc = logger.getLoggerNone(name)
	case "release":
		logger.errorFunc = logger.getLoggerError(name)
		logger.infoFunc = logger.getLoggerNone(name)
		logger.debugFunc = logger.getLoggerNone(name)
	default:
		os.Setenv("LOG_MODE", "release")
		return GetLogger(name)
	}
	return logger
}

// GetLogMode GetLogMode
func GetLogMode() (mode string) {
	if mode = strings.ToLower(os.Getenv("LOG_MODE")); mode == "" {
		mode = "release"
	}
	return
}

// PanicIf print error log and panic
func (o *Logger) PanicIf(err error) {
	if err != nil {
		o.errorFunc(err)
		panic(err)
	}
}

// Error print error log
func (o *Logger) Error(params ...interface{}) {
	o.errorFunc(params...)
}

// Info print info log
func (o *Logger) Info(params ...interface{}) {
	o.infoFunc(params...)
}

// Debug print debug log
func (o *Logger) Debug(params ...interface{}) {
	o.debugFunc(params...)
}

func (o *Logger) getLoggerInfo(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", green, "INFO", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func (o *Logger) getLoggerError(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", red, "ERROR", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func (o *Logger) getLoggerDebug(name string) func(params ...interface{}) {
	var logger = log.New(os.Stderr, fmt.Sprintf("%s[%s] - %s>>%s ", white, "DEBUG", name, reset), log.LstdFlags)
	return func(params ...interface{}) {
		if len(params) > 0 {
			logger.Println(params...)
		}
	}
}

func (o *Logger) getLoggerNone(name string) func(params ...interface{}) {
	return func(params ...interface{}) {
	}
}
