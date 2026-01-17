package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func New(logLevel string) *logrus.Logger {
	log := logrus.New()

	log.SetFormatter(&logrus.JSONFormatter{})

	log.SetOutput(os.Stdout)

	// Уровень логирования
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Уровень по умолчанию, если передан некорректный
	}
	log.SetLevel(level)
	return log
}
