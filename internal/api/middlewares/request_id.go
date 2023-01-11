package middlewares

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/log"
)

type (
	// RequestIDConfig defines the config for RequestID middleware.
	RequestIDConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// Generator defines a function to generate an ID.
		Generator func() string

		// RequestIDHandler defines a function which is executed for a request id.
		RequestIDHandler func(echo.Context, string)

		// TargetHeader defines what header to look for to populate the id
		TargetHeader string
	}
)

var (
	// DefaultRequestIDConfig is the default RequestID middleware config.
	DefaultRequestIDConfig = RequestIDConfig{
		Skipper:      DefaultSkipper,
		Generator:    generator,
		TargetHeader: echo.HeaderXRequestID,
	}
)

// RequestID returns a X-Request-ID middleware.
func RequestID() echo.MiddlewareFunc {
	return RequestIDWithConfig(DefaultRequestIDConfig)
}

// RequestIDWithConfig returns a X-Request-ID middleware with config.
func RequestIDWithConfig(config RequestIDConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultRequestIDConfig.Skipper
	}
	if config.Generator == nil {
		config.Generator = generator
	}
	if config.TargetHeader == "" {
		config.TargetHeader = echo.HeaderXRequestID
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			rid := config.Generator()
			c.Request().Header.Set(config.TargetHeader, rid) // prevent reading the custom id by the logger middleware
			c.Response().Header().Set(config.TargetHeader, rid)
			if config.RequestIDHandler != nil {
				config.RequestIDHandler(c, rid)
			}

			return next(c)
		}
	}
}

func generator() string {
	u, err := uuid.NewRandom()
	if err != nil {
		log.Logger.Api.Errorf("uuid.NewRandom: %s", err)
	}

	return u.String()
}
