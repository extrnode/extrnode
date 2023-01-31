package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"
)

// RequestID returns a X-Request-ID middleware.
func RequestDurationMiddleware(contextKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(contextKey, time.Now())
			return next(c)
		}
	}
}
