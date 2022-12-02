package api

import (
	"context"
	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
)

type api struct {
	port    uint64
	router  *echo.Echo
	storage storage.Storage

	waitGroup *sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewAPI(cfg config.Config) (*api, error) {
	s, err := storage.New(cfg.Postgres)
	if err != nil {
		return nil, errors.Wrap(err, "storage init")
	}

	var wg sync.WaitGroup
	ctx, cancelFunc := context.WithCancel(context.TODO())

	api := &api{
		port:    uint64(cfg.API.Port),
		router:  echo.New(),
		storage: s,

		waitGroup: &wg,
		ctx:       ctx,
		ctxCancel: cancelFunc,
	}

	api.router.Use(middleware.Recover())
	api.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	apiGroup := api.router.Group("/api/v1")
	apiGroup.GET("/info", api.getInfo)

	return api, nil
}

func (a *api) Run() error {
	go func() {
		<-a.ctx.Done()
		err := a.router.Shutdown(context.Background())
		if err != nil {
			log.Logger.Api.Errorf("api shutdown error: %s", err.Error())
		}
	}()

	err := a.router.Start(fmt.Sprintf(":%d", a.port))
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (a *api) getInfo(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

func (a *api) Stop() error {
	a.ctxCancel()
	return nil
}

func (a *api) WaitGroup() *sync.WaitGroup {
	return a.waitGroup
}
