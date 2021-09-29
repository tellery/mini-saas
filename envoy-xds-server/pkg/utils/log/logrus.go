package log

import "github.com/sirupsen/logrus"

var (
	logger *logrus.Logger
)

func init() {
	logger = logrus.New()
	// logger.AddHook(NewSentryHook())
	logger.SetFormatter(&logrus.JSONFormatter{})
}

func Logger() *logrus.Logger {
	return logger
}
