package config

import (
	"fmt"

	"extrnode-be/internal/pkg/config_types"
)

type Config struct {
	UApi config_types.UserApiConfig
	PG   config_types.PostgresConfig
}

func (c Config) Validate() error {
	if err := c.UApi.Validate(); err != nil {
		return fmt.Errorf("user_api: %s", err)
	}
	if err := c.PG.Validate(); err != nil {
		return fmt.Errorf("postgres: %s", err)
	}

	return nil
}
