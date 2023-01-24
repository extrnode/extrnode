package api

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log2 "github.com/labstack/gommon/log"
	"github.com/patrickmn/go-cache"

	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/api/middlewares/proxy"
	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/metrics"
	"extrnode-be/internal/pkg/storage"
)

// holds swagger static web server content.
//
//go:embed swaggerui
var swaggerDist embed.FS

type api struct {
	apiPort       uint64
	metricsPort   uint64
	certData      []byte
	router        *echo.Echo
	metricsServer *echo.Echo
	storage       storage.PgStorage
	cache         *cache.Cache
	waitGroup     *sync.WaitGroup
	ctx           context.Context
	ctxCancel     context.CancelFunc

	supportedOutputFormats map[string]struct{}
	blockchainIDs          map[string]int
	apiPrivateKey          solana.PrivateKey
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

	s, err := storage.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("storage init: %s", err)
	}

	blockchainsMap, err := s.GetBlockchainsMap()
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainsMap: %s", err)
	}

	// TODO: get from config
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("NewRandomPrivateKey: %s", err)
	}

	a := &api{
		apiPort:       cfg.API.Port,
		metricsPort:   cfg.API.MetricsPort,
		router:        echo.New(),
		metricsServer: echo.New(),
		storage:       s,
		cache:         cache.New(cacheTTL, cacheTTL),

		waitGroup: &sync.WaitGroup{},
		ctx:       ctx,
		ctxCancel: cancelFunc,
		supportedOutputFormats: map[string]struct{}{
			jsonOutputFormat:    {},
			csvOutputFormat:     {},
			haproxyOutputFormat: {},
		},
		blockchainIDs: blockchainsMap,
		apiPrivateKey: privKey,
	}

	if cfg.API.CertFile != "" {
		a.certData, err = os.ReadFile(cfg.API.CertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to read certificate (%s): %s", cfg.API.CertFile, err)
		}
	}

	a.setupServer()

	err = a.initApiHandlers()
	if err != nil {
		return nil, fmt.Errorf("initApiHandlers: %s", err)
	}

	return a, nil
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
	a.metricsServer.Use(middleware.Recover())
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
	// Scrape metrics from Main Server
	a.metricsServer.Use(prom.HandlerFunc)
	// Setup metrics endpoint at another server
	prom.SetMetricsPath(a.metricsServer)

	metrics.InitStartTime()
}

func (a *api) initApiHandlers() error {
	a.router.Use(middleware.Recover())
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
	generalGroup.GET("/endpoints", a.endpointsHandler)
	generalGroup.GET("/stats", a.statsHandler)

	transport := proxy.NewProxyTransport()
	go a.updateProxyEndpoints(transport)

	const (
		reqMethodContextKey         = "req_method"
		reqBodyContextKey           = "req_body"
		resBodyContextKey           = "res_body"
		rpcErrorContextKey          = "res_err"
		proxyEndpointContextKey     = "proxy_host"
		proxyAttemptsContextKey     = "proxy_attempts"
		proxyResponseTimeContextKey = "proxy_time"
	)

	// proxy
	generalGroup.POST("/", nil,
		middlewares.RequestIDMiddleware(),
		middlewares.NewLoggerMiddleware(middlewares.LoggerContextConfig{
			ReqMethodContextKey:         reqMethodContextKey,
			ReqBodyContextKey:           reqBodyContextKey,
			ResBodyContextKey:           resBodyContextKey,
			RpcErrorContextKey:          rpcErrorContextKey,
			ProxyEndpointContextKey:     proxyEndpointContextKey,
			ProxyAttemptsContextKey:     proxyAttemptsContextKey,
			ProxyResponseTimeContextKey: proxyResponseTimeContextKey,
		}),
		middlewares.NewMetricsMiddleware(middlewares.MetricsContextConfig{
			ReqMethodContextKey:         reqMethodContextKey,
			RpcErrorContextKey:          rpcErrorContextKey,
			ProxyEndpointContextKey:     proxyEndpointContextKey,
			ProxyAttemptsContextKey:     proxyAttemptsContextKey,
			ProxyResponseTimeContextKey: proxyResponseTimeContextKey,
		}),
		middlewares.NewBodyDumpMiddleware(middlewares.BodyDumpContextConfig{
			ReqMethodContextKey: reqMethodContextKey,
			ReqBodyContextKey:   reqBodyContextKey,
			ResBodyContextKey:   resBodyContextKey,
			RpcErrorContextKey:  rpcErrorContextKey,
		}),
		proxy.NewProxyMiddleware(transport, proxy.ProxyContextConfig{
			ProxyEndpointContextKey:     proxyEndpointContextKey,
			ProxyAttemptsContextKey:     proxyAttemptsContextKey,
			ProxyResponseTimeContextKey: proxyResponseTimeContextKey,
		}),
	)

	// api docs
	generalGroup.StaticFS("/swagger", echo.MustSubFS(swaggerDist, "swaggerui"))

	return nil
}

func (a *api) Run() (err error) {
	addr := fmt.Sprintf(":%d", a.apiPort)
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
	if a.metricsPort == 0 {
		return nil
	}
	err = a.metricsServer.Start(fmt.Sprintf(":%d", a.metricsPort))
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

func (a *api) getEndpointsURLs(blockchain string) ([]*url.URL, error) {
	blockchainID, ok := a.blockchainIDs[blockchain]
	if !ok {
		return nil, fmt.Errorf("fail to get blockchainID")
	}
	isRpc := true

	endpoints, err := a.storage.GetEndpoints(blockchainID, maxLimit, &isRpc, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("GetEndpoints: %s", err)
	}

	// temp sort solution
	// TODO: sort by methods
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].SupportedMethods.AverageResponseTime() < endpoints[j].SupportedMethods.AverageResponseTime()
	})

	var urls []*url.URL
	for _, e := range endpoints {
		schema := "http://"
		if e.IsSsl {
			schema = "https://"
		}
		parsedUrl, err := url.Parse(fmt.Sprintf("%s%s", schema, e.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %s", err)
		}

		urls = append(urls, parsedUrl)
	}

	return urls, nil
}

func (a *api) updateProxyEndpoints(transport *proxy.ProxyTransport) {
	for {
		urls, err := a.getEndpointsURLs(solanaBlockchain)
		if err != nil {
			log.Logger.Api.Logger.Fatalf("Cannot get endpoints from db: %s", err.Error())
		}

		transport.UpdateTargets(urls)

		time.Sleep(endpointsReloadInterval)
	}
}
