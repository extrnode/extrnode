package config

import (
	"fmt"

	"extrnode-be/internal/pkg/config_types"
)

type Config struct {
	Proxy config_types.ProxyConfig
	SL    config_types.SQLiteConfig
	CH    config_types.ClickhouseConfig
}

func (c Config) Validate() error {
	if err := c.Proxy.Validate(); err != nil {
		return fmt.Errorf("proxy: %s", err)
	}
	if err := c.SL.Validate(); err != nil {
		return fmt.Errorf("sqlite: %s", err)
	}

	return nil
}
