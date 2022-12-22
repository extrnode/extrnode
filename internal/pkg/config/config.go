package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// struct field names are used for env variable names. Edit with care
type Config struct {
	Scanner ScannerConfig
	API     ApiConfig
	PG      PostgresConfig
	//Metrics  MetricsConfig
}

type ScannerConfig struct {
	ThreadsNum int `required:"true" split_words:"true"`
}

type ApiConfig struct {
	Port int `required:"true" split_words:"true"`
}

type PostgresConfig struct {
	Host           string `required:"true" split_words:"true"`
	Port           uint64 `required:"true" split_words:"true"`
	User           string `required:"true" split_words:"true"`
	Pass           string `required:"true" split_words:"true"`
	DB             string `required:"true" split_words:"true"`
	MigrationsPath string `required:"true" split_words:"true"`
}

type MetricsConfig struct {
	IsEnabled bool `required:"true" split_words:"true"`
	Port      int  `required:"true" split_words:"true"`
}

func LoadFile(envFile string) (c Config, err error) {
	if envFile != "" {
		err = godotenv.Load(envFile)
		if err != nil {
			return c, fmt.Errorf("godotenv.Load: %s", err)
		}
	}

	err = envconfig.Process("", &c)
	if err != nil {
		return c, fmt.Errorf("envconfig.Process: %s", err)
	}

	err = c.validate()
	if err != nil {
		return c, fmt.Errorf("validate %s: %s", envFile, err)
	}

	return c, nil
}
