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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/patrickmn/go-cache"

	"extrnode-be/internal/pkg/storage/postgres"
	echo2 "extrnode-be/internal/pkg/util/echo"

	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/sqlite"
)

// holds swagger static web server content.
//
//go:embed swaggerui
var swaggerDist embed.FS

type api struct {
	conf      config.ApiConfig
	certData  []byte
	router    *echo.Echo
	slStorage sqlite.Storage
	pgStorage postgres.Storage
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

	cacheTTL = 5 * time.Minute

	serverShutdownTimeout = 10 * time.Second
)

func NewAPI(cfg config.Config) (*api, error) {
	// increase uuid generation productivity
	//uuid.EnableRandPool()
	ctx, cancelFunc := context.WithCancel(context.Background())

	slStorage, err := sqlite.New(ctx, cfg.SL)
	if err != nil {
		return nil, fmt.Errorf("SL storage init: %s", err)
	}

	pgStorage, err := postgres.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("PG storage init: %s", err)
	}

	blockchainsMap, err := slStorage.GetBlockchainsMap()
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainsMap: %s", err)
	}

	// TODO: get from config
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("NewRandomPrivateKey: %s", err)
	}

	a := &api{
		conf:      cfg.API,
		router:    echo.New(),
		slStorage: slStorage,
		pgStorage: pgStorage,
		cache:     cache.New(cacheTTL, cacheTTL),

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

	echo2.SetupServer(a.router)

	err = a.initApiHandlers()

	return a, err
}

func (a *api) initApiHandlers() error {
	echo2.InitHandlersStart(a.router)

	a.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// public
	a.router.GET("/endpoints", a.endpointsHandler)
	a.router.GET("/stats", a.statsHandler)

	// api docs
	a.router.StaticFS("/swagger", echo.MustSubFS(swaggerDist, "swaggerui"))

	// protected
	aMw, err := middlewares.NewAuthMiddleware(a.ctx, a.conf)
	if err != nil {
		return fmt.Errorf("NewAuthMiddleware: %s", err)
	}
	protectedGroup := a.router.Group("", aMw.LoadUser)
	protectedGroup.GET("/api_token", a.apiTokenHandler)

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

func (a *api) Stop() error {
	ctx, cancel := context.WithTimeout(a.ctx, serverShutdownTimeout)
	defer cancel()

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
