package log

import (
	log "github.com/sirupsen/logrus"
)

type typedLog struct {
	Scanner    *log.Entry
	BatchSaver *log.Entry
	Api        *log.Entry
}

var (
	Logger *typedLog
)

// Init logger on start
func init() {
	Logger = &typedLog{
		Scanner:    log.WithFields(log.Fields{"module": "scanner"}),
		BatchSaver: log.WithFields(log.Fields{"module": "batch_saver"}),
		Api:        log.WithFields(log.Fields{"module": "api"}),
	}
}

func Setup(lvl string) error {
	logLevel, err := log.ParseLevel(lvl)
	if err != nil {
		return err
	}

	// log format
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.SetLevel(logLevel)
	return nil
}
