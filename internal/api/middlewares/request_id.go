package middlewares

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/log"
)

// RequestID returns a X-Request-ID middleware.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rid, err := uuid.NewRandom()
			if err != nil {
				log.Logger.Api.Errorf("uuid.NewRandom: %s", err)
			}

			c.Request().Header.Set(echo.HeaderXRequestID, rid.String()) // prevent reading the custom id by the logger middleware
			c.Response().Header().Set(echo.HeaderXRequestID, rid.String())

			return next(c)
		}
	}
}
