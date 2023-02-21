package config

import (
	"fmt"

	"extrnode-be/internal/pkg/config_types"
)

type Config struct {
	Scanner config_types.ScannerConfig
	SL      config_types.SQLiteConfig
	CH      config_types.ClickhouseConfig
}

func (c Config) Validate() error {
	if err := c.SL.Validate(); err != nil {
		return fmt.Errorf("sqlite: %s", err)
	}
	if err := c.Scanner.Validate(); err != nil {
		return fmt.Errorf("scanner: %s", err)
	}

	return nil
}
