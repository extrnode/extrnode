package main

import (
	"flag"
	"time"

	"extrnode-be/internal/pkg/config_types"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/util"
	"extrnode-be/internal/scanner"
	"extrnode-be/internal/scanner/config"
)

const (
	waitTimeout = time.Minute
)

type flags struct {
	logLevel string
	envFile  string
}

// Setup flags
func getFlags() (f flags) {
	flag.StringVar(&f.logLevel, "log", "info", "log level [debug|info|warn|error|crit]")
	flag.StringVar(&f.envFile, "envFile", "", "path to .env file")
	flag.Parse()

	return
}

func main() {
	f := getFlags()
	err := log.Setup(f.logLevel)
	if err != nil {
		log.Logger.Scanner.Fatalf("Log setup: %s", err)
	}

	cfg, err := config_types.LoadFile[config.Config](f.envFile)
	if err != nil {
		log.Logger.Scanner.Fatalf("Config: %s", err)
	}

	log.Logger.Scanner.Info("Start service")

	app, err := scanner.NewScanner(cfg)
	if err != nil {
		log.Logger.Scanner.Fatalf("scanner error: %s", err)
	}

	// Run service
	go func() {
		err := app.Run()
		if err != nil {
			log.Logger.Scanner.Fatalf("Scanner: %s", err)
		}
	}()

	// Run nmap
	go func() {
		err := app.RunNmap()
		if err != nil {
			log.Logger.Scanner.Fatalf("NmapScan: %s", err)
		}
	}()

	go func() {
		err := app.CheckOutdatedNodes()
		if err != nil {
			log.Logger.Scanner.Fatalf("CheckOutdatedNodes: %s", err)
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
