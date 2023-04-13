package utils

import "github.com/sirupsen/logrus"

func InitlogrusLogger(logrusLogger *logrus.Logger) {

	logrusLogger.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05.000000", FullTimestamp: true})
	logrusLogger.SetLevel(logrus.DebugLevel)
}
