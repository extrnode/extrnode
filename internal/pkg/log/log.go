package log

import (
	log "github.com/sirupsen/logrus"
)

type typedLog struct {
	Scanner *log.Entry
	Api     *log.Entry
	Proxy   *log.Entry
}

var (
	Logger *typedLog
)

// Init logger on start
func init() {
	Logger = &typedLog{
		Scanner: log.WithFields(log.Fields{"module": "scanner"}),
		Api:     log.WithFields(log.Fields{"module": "api"}),
		Proxy:   log.WithFields(log.Fields{"module": "proxy"}),
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
