package main

import (
	"extrnode-be/internal/api"
	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/util"
	"flag"
	"time"
)

const (
	waitTimeout = time.Second * 10
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
		log.Logger.Scanner.Fatalf("Log setup: %s", err.Error())
	}

	cfg, err := config.LoadFile(f.config)
	if err != nil {
		log.Logger.Scanner.Fatalf("Config error: %s", err)
	}

	app, err := api.NewAPI(cfg)
	if err != nil {
		log.Logger.Api.Fatalf("NewAPI: %s", err.Error())
	}

	// API
	go func() {
		if err := app.Run(); err != nil {
			log.Logger.Api.Fatalf("API: %s", err.Error())
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
