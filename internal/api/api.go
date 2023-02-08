package api

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log2 "github.com/labstack/gommon/log"
	"github.com/patrickmn/go-cache"

	"extrnode-be/internal/api/log_collector"
	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/api/middlewares/proxy"
	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/metrics"
	"extrnode-be/internal/pkg/storage/clickhouse"
	"extrnode-be/internal/pkg/storage/postgres"
)

// holds swagger static web server content.
//
//go:embed swaggerui
var swaggerDist embed.FS

type api struct {
	conf          config.ApiConfig
	metricsPort   uint64
	certData      []byte
	router        *echo.Echo
	metricsServer *echo.Echo
	pgStorage     postgres.Storage
	chStorage     clickhouse.Storage
	cache         *cache.Cache
	waitGroup     *sync.WaitGroup
	ctx           context.Context
	ctxCancel     context.CancelFunc

	supportedOutputFormats map[string]struct{}
	blockchainIDs          map[string]int
	apiPrivateKey          solana.PrivateKey
	logCollector *log_collector.Collector
}

const (
	jsonOutputFormat    = "json"
	csvOutputFormat     = "csv"
	haproxyOutputFormat = "haproxy"

	cacheTTL        = 5 * time.Minute
	apiReadTimeout  = 5 * time.Second
	apiWriteTimeout = 30 * time.Second

	serverShutdownTimeout = 10 * time.Second

	endpointsReloadInterval = 5 * time.Minute
)

func NewAPI(cfg config.Config) (*api, error) {
	// increase uuid generation productivity
	//uuid.EnableRandPool()
	ctx, cancelFunc := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("PG storage init: %s", err)
	}
	chStorage, err := clickhouse.New(cfg.CH.DSN)
	if err != nil {
		return nil, fmt.Errorf("CH storage init: %s", err)
	}

	blockchainsMap, err := pgStorage.GetBlockchainsMap()
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainsMap: %s", err)
	}

	// TODO: get from config
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("NewRandomPrivateKey: %s", err)
	}

	a := &api{
		conf:          cfg.API,
		router:        echo.New(),
		metricsServer: echo.New(),
		pgStorage:     pgStorage,
		chStorage:     chStorage,
		cache:         cache.New(cacheTTL, cacheTTL),

		waitGroup: &sync.WaitGroup{},
		ctx:       ctx,
		ctxCancel: cancelFunc,
		supportedOutputFormats: map[string]struct{}{
			jsonOutputFormat:    {},
			csvOutputFormat:     {},
			haproxyOutputFormat: {},
		},
		blockchainIDs:   blockchainsMap,
		apiPrivateKey:   privKey,
		logCollector: log_collector.NewCollector(ctx, chStorage),
	}

	if cfg.API.CertFile != "" {
		a.certData, err = os.ReadFile(cfg.API.CertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to read certificate (%s): %s", cfg.API.CertFile, err)
		}
	}

	a.setupServer()

	err = a.initApiHandlers()

	go a.logCollector.StartStatSaver()

	return a, err
}

func (a *api) setupServer() {
	a.router.Server.ReadTimeout = apiReadTimeout
	a.router.Server.WriteTimeout = apiWriteTimeout + 2*time.Second // must be greater than apiWriteTimeout, which used for timeout middleware
	a.router.Logger.SetLevel(log2.OFF)

	a.metricsServer.Server.ReadTimeout = apiReadTimeout
	a.metricsServer.Server.WriteTimeout = apiWriteTimeout + 2*time.Second // must be greater than apiWriteTimeout, which used for timeout middleware
	a.metricsServer.Logger.SetLevel(log2.OFF)
}

func (a *api) initMetrics() {
	a.metricsServer.HideBanner = true
	a.metricsServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc:    logPanic,
	}))
	a.metricsServer.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogMethod: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				log.Logger.Api.Errorf("metrics: code %d method %s: %s", v.Status, v.Method, v.Error)
			}
			return nil
		},
	}))

	prom := prometheus.NewPrometheus("extrnode", nil, metrics.MetricList())
	// Setup metrics endpoint at another server
	prom.SetMetricsPath(a.metricsServer)

	metrics.InitStartTime()
}

func (a *api) initApiHandlers() error {
	a.router.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc:    logPanic,
	}))
	a.router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&middlewares.CustomContext{
				Context: c,
			})
		}
	})
	a.router.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request Timeout",
		Timeout:      apiWriteTimeout,
	}))

	// prometheus metrics
	a.initMetrics()

	// general rate limit
	a.router.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // req per second

	generalGroup := a.router.Group("", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	// public
	generalGroup.GET("/endpoints", a.endpointsHandler)
	generalGroup.GET("/stats", a.statsHandler)

	// protected
	aMw, err := middlewares.NewAuthMiddleware(a.ctx, a.conf)
	if err != nil {
		return fmt.Errorf("NewAuthMiddleware: %s", err)
	}
	protectedGroup := generalGroup.Group("", aMw.LoadUser)
	protectedGroup.GET("/api_token", a.apiTokenHandler)

	transport, err := proxy.NewProxyTransport(false, a.conf.FailoverEndpoints)
	if err != nil {
		return err
	}

	go a.updateProxyEndpoints(transport)

	// proxy
	generalGroup.POST("/", nil,
		middlewares.RequestDurationMiddleware(),
		middlewares.RequestIDMiddleware(),
		middlewares.NewLoggerMiddleware(a.logCollector.AddStat),
		middlewares.NewMetricsMiddleware(),
		middlewares.NewValidatorMiddleware(),
		proxy.NewProxyMiddleware(transport),
	)

	// api docs
	generalGroup.StaticFS("/swagger", echo.MustSubFS(swaggerDist, "swaggerui"))

	return nil
}

func (a *api) Run() (err error) {
	addr := fmt.Sprintf(":%d", a.conf.Port)
	if len(a.certData) != 0 {
		err = a.router.StartTLS(addr, a.certData, a.certData)
	} else {
		err = a.router.Start(addr)
	}
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (a *api) RunMetrics() (err error) {
	if a.conf.MetricsPort == 0 {
		return nil
	}
	err = a.metricsServer.Start(fmt.Sprintf(":%d", a.conf.MetricsPort))
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (a *api) Stop() error {
	ctx, cancel := context.WithTimeout(a.ctx, serverShutdownTimeout)
	defer cancel()

	go a.metricsServer.Shutdown(ctx)
	err := a.router.Shutdown(ctx)
	if err != nil {
		log.Logger.Api.Errorf("router.Shutdown: %s", err)
	}
	a.ctxCancel()

	return nil
}

func (a *api) WaitGroup() *sync.WaitGroup {
	return a.waitGroup
}
