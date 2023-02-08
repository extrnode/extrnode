package config

import (
	"encoding/json"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// struct field names are used for env variable names. Edit with care
type Config struct {
	Scanner ScannerConfig
	API     ApiConfig
	PG      PostgresConfig
	CH      ClickhouseConfig
}

type ScannerConfig struct {
	ThreadsNum int `required:"true" split_words:"true"`
}

type ApiConfig struct {
	Port              uint64          `required:"true" split_words:"true"`
	MetricsPort       uint64          `required:"false" split_words:"true"`
	CertFile          string          `required:"false" split_words:"true"`
	FailoverEndpoints FailoverTargets `required:"false" split_words:"true"`
}

type PostgresConfig struct {
	Host           string `required:"true" split_words:"true"`
	Port           uint64 `required:"true" split_words:"true"`
	User           string `required:"true" split_words:"true"`
	Pass           string `required:"true" split_words:"true"`
	DB             string `required:"true" split_words:"true"`
	MigrationsPath string `required:"true" split_words:"true"`
}

type ClickhouseConfig struct {
	DSN string `required:"true" split_words:"true"`
}

// AddTarget adds an upstream target to the list.
type (
	FailoverTargets []struct {
		Url            string
		ReqLimitHourly uint64
	}
)

func (f *FailoverTargets) Decode(value string) error {
	if len(value) == 0 {
		return nil
	}

	return json.Unmarshal([]byte(value), &f)
}

func LoadFile(envFile string) (c Config, err error) {
	if envFile != "" {
		err = godotenv.Load(envFile)
		if err != nil {
			return c, fmt.Errorf("godotenv.Load (%s): %s", envFile, err)
		}
	}

	err = envconfig.Process("", &c)
	if err != nil {
		return c, fmt.Errorf("envconfig.Process: %s", err)
	}

	err = c.validate()
	if err != nil {
		return c, fmt.Errorf("validate: %s", err)
	}

	return c, nil
}
