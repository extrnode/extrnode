package user_api

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

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/postgres"
	echo2 "extrnode-be/internal/pkg/util/echo"
	"extrnode-be/internal/user_api/middlewares"
)

// holds swagger static web server content.
//
//go:embed swaggerui
var swaggerDist embed.FS

type userApi struct {
	conf      config.UserApiConfig
	certData  []byte
	router    *echo.Echo
	pgStorage postgres.Storage
	cache     *cache.Cache
	waitGroup *sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc

	apiPrivateKey solana.PrivateKey
}

const (
	cacheTTL = 5 * time.Minute

	serverShutdownTimeout = 10 * time.Second
)

func NewAPI(cfg config.Config) (*userApi, error) {
	// increase uuid generation productivity
	//uuid.EnableRandPool()
	ctx, cancelFunc := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("PG storage init: %s", err)
	}

	// TODO: get from config
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("NewRandomPrivateKey: %s", err)
	}

	a := &userApi{
		conf:      cfg.UApi,
		router:    echo.New(),
		pgStorage: pgStorage,
		cache:     cache.New(cacheTTL, cacheTTL),

		waitGroup:     &sync.WaitGroup{},
		ctx:           ctx,
		ctxCancel:     cancelFunc,
		apiPrivateKey: privKey,
	}

	if cfg.UApi.CertFile != "" {
		a.certData, err = os.ReadFile(cfg.UApi.CertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to read certificate (%s): %s", cfg.UApi.CertFile, err)
		}
	}

	echo2.SetupServer(a.router)

	err = a.initApiHandlers()

	return a, err
}

func (a *userApi) initApiHandlers() error {
	echo2.InitHandlersStart(a.router)

	a.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

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

func (a *userApi) Run() (err error) {
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

func (a *userApi) Stop() error {
	ctx, cancel := context.WithTimeout(a.ctx, serverShutdownTimeout)
	defer cancel()

	err := a.router.Shutdown(ctx)
	if err != nil {
		log.Logger.UserApi.Errorf("router.Shutdown: %s", err)
	}
	a.ctxCancel()

	return nil
}

func (a *userApi) WaitGroup() *sync.WaitGroup {
	return a.waitGroup
}
