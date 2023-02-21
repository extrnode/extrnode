package config_types

import (
	"encoding/json"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// struct field names are used for env variable names. Edit with care
type (
	ScannerConfig struct {
		ThreadsNum int    `required:"true" split_words:"true"`
		Hostname   string `required:"true" split_words:"true"`
	}
	ScannerApiConfig struct {
		Port     uint64 `required:"true" split_words:"true"`
		CertFile string `required:"false" split_words:"true"`
	}
	ProxyConfig struct {
		Port              uint64          `required:"true" split_words:"true"`
		MetricsPort       uint64          `required:"false" split_words:"true"`
		CertFile          string          `required:"false" split_words:"true"`
		FailoverEndpoints FailoverTargets `required:"false" split_words:"true"`
	}
	UserApiConfig struct {
		Port             uint64 `required:"true" split_words:"true"`
		CertFile         string `required:"false" split_words:"true"`
		FirebaseFilePath string `required:"true" split_words:"true"`
	}
)

// struct field names are used for env variable names. Edit with care
type (
	PostgresConfig struct {
		Host           string `required:"true" split_words:"true"`
		Port           uint64 `required:"true" split_words:"true"`
		User           string `required:"true" split_words:"true"`
		Pass           string `required:"true" split_words:"true"`
		DB             string `required:"true" split_words:"true"`
		MigrationsPath string `required:"true" split_words:"true"`
	}
	SQLiteConfig struct {
		DBPath         string `required:"true" split_words:"true"`
		MigrationsPath string `required:"true" split_words:"true"`
	}
	ClickhouseConfig struct {
		DSN string `required:"true" split_words:"true"`
	}
)

type FailoverTargets []struct {
	Url            string
	ReqLimitHourly uint64
}

func (f *FailoverTargets) Decode(value string) error {
	if len(value) == 0 {
		return nil
	}

	return json.Unmarshal([]byte(value), &f)
}

type PossibleConfig interface {
	Validate() error
}

func LoadFile[T PossibleConfig](envFile string) (c T, err error) {
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

	err = c.Validate()
	if err != nil {
		return c, fmt.Errorf("validate: %s", err)
	}

	return c, nil
}
