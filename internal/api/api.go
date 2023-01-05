package api

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/patrickmn/go-cache"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

// holds swagger static web server content.
//
//go:embed swaggerui
var swaggerDist embed.FS

type api struct {
	port    uint64
	router  *echo.Echo
	storage storage.PgStorage
	cache   *cache.Cache

	waitGroup              *sync.WaitGroup
	ctx                    context.Context
	ctxCancel              context.CancelFunc
	supportedOutputFormats map[string]struct{}
	blockchainIDs          map[string]int
}

const (
	jsonOutputFormat    = "json"
	csvOutputFormat     = "csv"
	haproxyOutputFormat = "haproxy"

	cacheTTL = 5 * time.Minute
)

func NewAPI(cfg config.Config) (*api, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s, err := storage.New(ctx, cfg.PG)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("storage init: %s", err)
	}

	blockchainsMap, err := s.GetBlockchainsMap()
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("GetBlockchainsMap: %s", err)
	}

	a := &api{
		port:    uint64(cfg.API.Port),
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
	}

	a.initApiHandlers()

	return a, nil
}

func (a *api) initApiHandlers() {
	a.router.Use(middleware.Recover())
	a.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	a.router.GET("/endpoints", a.getEndpointsHandler)
	a.router.GET("/stats", a.getStatsHandler)

	// api docs
	a.router.StaticFS("/swagger", echo.MustSubFS(swaggerDist, "swaggerui"))
}

func (a *api) Run() error {
	go func() {
		<-a.ctx.Done()
		err := a.router.Shutdown(context.Background())
		if err != nil {
			log.Logger.Api.Errorf("api shutdown error: %s", err)
		}
	}()

	err := a.router.Start(fmt.Sprintf(":%d", a.port))
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
