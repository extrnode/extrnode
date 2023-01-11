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
	"github.com/patrickmn/go-cache"

	"extrnode-be/internal/api/middlewares"
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
	certData  []byte
	router    *echo.Echo
	storage   storage.PgStorage
	cache     *cache.Cache
	waitGroup *sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc

	supportedOutputFormats map[string]struct{}
	blockchainIDs          map[string]int
	apiPrivateKey          solana.PrivateKey
}

const (
	jsonOutputFormat    = "json"
	csvOutputFormat     = "csv"
	haproxyOutputFormat = "haproxy"

	endpointHeader    = "X-ENDPOINT"
	signatureHeader   = "X-SIGNATURE"
	elapsedTimeHeader = "X-ELAPSED-TIME"

	cacheTTL                     = 5 * time.Minute
	apiReadTimeout               = 5 * time.Second
	apiWriteTimeout              = 30 * time.Second
	customTransportDialerTimeout = 2 * time.Second

	apiPort = 8000
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
		router:  echo.New(),
		storage: s,
		cache:   cache.New(cacheTTL, cacheTTL),

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

	a.router.Server.ReadTimeout = apiReadTimeout
	a.router.Server.WriteTimeout = apiWriteTimeout + 2*time.Second // must be greater than apiWriteTimeout, which used for timeout middleware

	err = a.initApiHandlers()
	if err != nil {
		return nil, fmt.Errorf("initApiHandlers: %s", err)
	}

	return a, nil
}

func (a *api) initApiHandlers() error {
	a.router.Use(middleware.Recover())
	a.router.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request Timeout",
		Timeout:      apiWriteTimeout,
	}))

	// prometheus metrics
	prometheus.NewPrometheus("extrnode", nil, metrics.MetricList()).Use(a.router)
	metrics.InitStartTime()

	// general rate limit
	a.router.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // req per second

	generalGroup := a.router.Group("", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	generalGroup.GET("/endpoints", a.endpointsHandler)
	generalGroup.GET("/stats", a.statsHandler)

	// api docs
	generalGroup.StaticFS("/swagger", echo.MustSubFS(swaggerDist, "swaggerui"))

	// chains
	chainsGroup := a.router.Group("", middleware.Logger(), middlewares.RequestID())
	err := a.solanaProxyHandler(chainsGroup)
	if err != nil {
		return fmt.Errorf("solanaProxyHandler: %s", err)
	}

	return nil
}

func (a *api) Run() (err error) {
	go func() {
		<-a.ctx.Done()
		err := a.router.Shutdown(context.Background())
		if err != nil {
			log.Logger.Api.Errorf("api shutdown error: %s", err)
		}
	}()

	addr := fmt.Sprintf(":%d", apiPort)
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

func (a *api) Stop() error {
	a.ctxCancel()
	return nil
}

func (a *api) WaitGroup() *sync.WaitGroup {
	return a.waitGroup
}
