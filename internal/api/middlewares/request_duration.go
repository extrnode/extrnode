package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"
)

// RequestID returns a X-Request-ID middleware.
func RequestDurationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := c.(*CustomContext)
			cc.SetReqDuration(time.Now())
			return next(c)
		}
	}
}
