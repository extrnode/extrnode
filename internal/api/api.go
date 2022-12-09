package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

type api struct {
	port    uint64
	router  *echo.Echo
	storage storage.PgStorage

	waitGroup              *sync.WaitGroup
	ctx                    context.Context
	ctxCancel              context.CancelFunc
	supportedOutputFormats map[string]struct{}
}

const (
	jsonOutputFormat    = "json"
	csvOutputFormat     = "csv"
	haproxyOutputFormat = "haproxy"
)

func NewAPI(cfg config.Config) (*api, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s, err := storage.New(ctx, cfg.Postgres)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("storage init: %s", err)
	}

	api := &api{
		port:    uint64(cfg.API.Port),
		router:  echo.New(),
		storage: s,

		waitGroup: &sync.WaitGroup{},
		ctx:       ctx,
		ctxCancel: cancelFunc,
		supportedOutputFormats: map[string]struct{}{
			jsonOutputFormat:    {},
			csvOutputFormat:     {},
			haproxyOutputFormat: {},
		},
	}

	api.initApiHandlers()

	return api, nil
}

func (a *api) initApiHandlers() {
	a.router.Use(middleware.Recover())
	a.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	apiGroup := a.router.Group("/api/v1")
	apiGroup.GET("/info", a.getInfo)
	apiGroup.GET("/endpoints", a.getEndpoints)
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
