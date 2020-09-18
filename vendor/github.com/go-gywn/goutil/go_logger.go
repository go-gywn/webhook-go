package goutil

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var logging = strings.ToLower(os.Getenv("LOGGING"))

// GetLogger get logger
func GetLogger() (log *logrus.Logger) {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		// DisableLevelTruncation: true,
		// DisableTimestamp: true,
		FullTimestamp:    true,
		QuoteEmptyFields: true,
		// TimestampFormat:  "2006-01-02T15:04:05.999-0700",
		TimestampFormat: "15:04:05",
	})

	switch logging {
	case "trace":
		log.Level = logrus.TraceLevel
	case "debug":
		log.Level = logrus.DebugLevel
	case "info":
		log.Level = logrus.InfoLevel
	case "test":
		log.Level = logrus.InfoLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	case "release":
		log.Level = logrus.ErrorLevel
	default:
		log.Level = logrus.ErrorLevel
	}
	return
}

// GinMode get gin mode
func GinMode() string {
	switch logging {
	case gin.TestMode:
	case gin.DebugMode:
	case gin.ReleaseMode:
	default:
		return gin.ReleaseMode
	}
	return logging
}
