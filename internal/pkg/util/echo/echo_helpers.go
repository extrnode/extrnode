package echo

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log2 "github.com/labstack/gommon/log"

	"extrnode-be/internal/pkg/log"
)

const (
	apiReadTimeout  = 5 * time.Second
	apiWriteTimeout = 30 * time.Second
)

func InitHandlersStart(router *echo.Echo) {
	router.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc:    LogPanic,
	}))
	router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&CustomContext{
				Context: c,
			})
		}
	})
	router.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request Timeout",
		Timeout:      apiWriteTimeout,
	}))

	// general rate limit
	router.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // req per second
}

func SetupServer(router *echo.Echo) {
	router.Server.ReadTimeout = apiReadTimeout
	router.Server.WriteTimeout = apiWriteTimeout + 2*time.Second // must be greater than apiWriteTimeout, which used for timeout middleware
	router.Logger.SetLevel(log2.OFF)
}

func LogPanic(c echo.Context, err error, stack []byte) error {
	log.Logger.Proxy.Errorf("PANIC RECOVER: %s %s", err, strconv.Quote(string(stack)))
	return nil
}
