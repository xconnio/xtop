package xtop

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var log = logrus.New()

const LogPath = "/tmp/xtop.log"

func SetupLogger(path string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("cannot open log file %s: %w", path, err)
	}

	log.Out = f
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05-07:00",
		DisableQuote:    true,
	}
	return nil
}
