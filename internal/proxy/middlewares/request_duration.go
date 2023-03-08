package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"

	echo2 "extrnode-be/internal/pkg/util/echo"
)

// RequestID returns a X-Request-ID middleware.
func RequestDurationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := c.(*echo2.CustomContext)
			cc.SetReqDuration(time.Now())
			return next(c)
		}
	}
}
