package internal

import "github.com/sirupsen/logrus"

func SetupLogging() error {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return nil
}
