package main

import (
	"flag"
	"time"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/util"
	"extrnode-be/internal/scanner"
)

const (
	waitTimeout = time.Minute
)

type flags struct {
	logLevel string
	config   string
}

// Setup flags
func getFlags() (f flags) {
	flag.StringVar(&f.logLevel, "log", "debug", "log level [debug|info|warn|error|crit]")
	flag.StringVar(&f.config, "conf", "./config.yml", "path to .env file")
	flag.Parse()

	return
}

func main() {
	f := getFlags()
	err := log.Setup(f.logLevel)
	if err != nil {
		log.Logger.Scanner.Fatalf("Log setup: %s", err)
	}

	cfg, err := config.LoadFile(f.config)
	if err != nil {
		log.Logger.Scanner.Fatalf("Config error: %s", err)
	}

	app, err := scanner.NewScanner(cfg)
	if err != nil {
		log.Logger.Scanner.Fatalf("scanner error: %s", err)
	}

	// Metrics
	// if conf.Metrics.IsEnabled {
	// 	err = metrics.StartHTTP(conf.Metrics.Port)
	// 	if err != nil {
	// 		log.Logger.Scanner.Fatalf("Metrics: %s", err)
	// 	}
	// }

	// Run service
	go func() {
		err := app.Run()
		if err != nil {
			log.Logger.Scanner.Fatalf("Scanner: %s", err)
		}
	}()

	// Termination handler.
	util.GracefulStop(app.WaitGroup(), waitTimeout, func() {
		err = app.Stop()
		if err != nil {
			log.Logger.Scanner.Errorf(err.Error())
		}
	})
}
