package config

import (
	"fmt"

	"extrnode-be/internal/pkg/config_types"
)

type Config struct {
	SApi config_types.ScannerApiConfig
	SL   config_types.SQLiteConfig
}

func (c Config) Validate() error {
	if err := c.SApi.Validate(); err != nil {
		return fmt.Errorf("scanner_api: %s", err)
	}
	if err := c.SL.Validate(); err != nil {
		return fmt.Errorf("sqlite: %s", err)
	}

	return nil
}
