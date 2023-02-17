package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/metrics"
	"extrnode-be/internal/pkg/storage/clickhouse"
	"extrnode-be/internal/pkg/storage/clickhouse/delayed_insertion"
	"extrnode-be/internal/pkg/storage/postgres"
	echo2 "extrnode-be/internal/pkg/util/echo"
	"extrnode-be/internal/proxy/middlewares"
)

const (
	solanaBlockchain = "solana"
)

type proxy struct {
	certData      []byte
	proxyPort     uint64
	metricsPort   uint64
	router        *echo.Echo
	metricsServer *echo.Echo
	pgStorage     postgres.Storage
	waitGroup     *sync.WaitGroup
	ctx           context.Context
	ctxCancel     context.CancelFunc

	blockchainIDs   map[string]int
	failoverTargets config.FailoverTargets

	statsCollector *delayed_insertion.Collector[clickhouse.Stat]
}

const (
	serverShutdownTimeout   = 10 * time.Second
	endpointsReloadInterval = 5 * time.Minute
	collectorInterval       = 10 * time.Second
)

func NewProxy(cfg config.Config) (*proxy, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("PG storage init: %s", err)
	}
	chStorage, err := clickhouse.New(cfg.CH.DSN, cfg.Scanner.Hostname)
	if err != nil {
		return nil, fmt.Errorf("CH storage init: %s", err)
	}

	blockchainsMap, err := pgStorage.GetBlockchainsMap()
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainsMap: %s", err)
	}

	p := &proxy{
		proxyPort:     cfg.Proxy.Port,
		metricsPort:   cfg.Proxy.MetricsPort,
		router:        echo.New(),
		metricsServer: echo.New(),
		pgStorage:     pgStorage,

		waitGroup:       &sync.WaitGroup{},
		ctx:             ctx,
		ctxCancel:       cancelFunc,
		blockchainIDs:   blockchainsMap,
		failoverTargets: cfg.Proxy.FailoverEndpoints,

		statsCollector: delayed_insertion.New[clickhouse.Stat](ctx, cfg, chStorage, collectorInterval),
	}

	if cfg.Proxy.CertFile != "" {
		p.certData, err = os.ReadFile(cfg.Proxy.CertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to read certificate (%s): %s", cfg.Proxy.CertFile, err)
		}
	}
	p.setupServer()

	err = p.initProxyHandlers()

	return p, err
}

func (p *proxy) setupServer() {
	echo2.SetupServer(p.router)
	echo2.SetupServer(p.metricsServer)
}

func (p *proxy) initMetrics() {
	p.metricsServer.HideBanner = true
	p.metricsServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc:    echo2.LogPanic,
	}))
	p.metricsServer.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogMethod: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				log.Logger.Proxy.Errorf("metrics: code %d method %s: %s", v.Status, v.Method, v.Error)
			}
			return nil
		},
	}))

	prom := prometheus.NewPrometheus("extrnode", nil, metrics.MetricList())
	// Setup metrics endpoint at another server
	prom.SetMetricsPath(p.metricsServer)

	metrics.InitStartTime()
}

func (p *proxy) initProxyHandlers() error {
	echo2.InitHandlersStart(p.router)

	// forked cors middleware
	p.router.Use(middlewares.CORSWithConfig(middlewares.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	// prometheus metrics
	p.initMetrics()

	scannedMethodList, err := p.getScannedMethods()
	if err != nil {
		return fmt.Errorf("getScannedMethods: %s", err)
	}

	transport, err := middlewares.NewProxyTransport(false, p.failoverTargets, scannedMethodList)
	if err != nil {
		return fmt.Errorf("NewProxyTransport: %s", err)
	}
	go p.updateProxyEndpoints(transport)

	// proxy
	p.router.POST("/", nil,
		middlewares.RequestDurationMiddleware(),
		middlewares.RequestIDMiddleware(),
		middlewares.NewLoggerMiddleware(p.statsCollector.Add),
		middlewares.NewMetricsMiddleware(),
		middlewares.NewValidatorMiddleware(),
		middlewares.NewProxyMiddleware(transport),
	)

	return nil
}

func (p *proxy) Run() (err error) {
	addr := fmt.Sprintf(":%d", p.proxyPort)
	if len(p.certData) != 0 {
		err = p.router.StartTLS(addr, p.certData, p.certData)
	} else {
		err = p.router.Start(addr)
	}

	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (p *proxy) RunMetrics() (err error) {
	if p.metricsPort == 0 {
		return nil
	}
	err = p.metricsServer.Start(fmt.Sprintf(":%d", p.metricsPort))
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (p *proxy) Stop() error {
	ctx, cancel := context.WithTimeout(p.ctx, serverShutdownTimeout)
	defer cancel()

	go p.metricsServer.Shutdown(ctx)
	err := p.router.Shutdown(ctx)
	if err != nil {
		log.Logger.Proxy.Errorf("router.Shutdown: %s", err)
	}
	p.ctxCancel()

	return nil
}

func (p *proxy) WaitGroup() *sync.WaitGroup {
	return p.waitGroup
}
