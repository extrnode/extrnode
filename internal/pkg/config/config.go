package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config is the top-level configuration for Prometheus's config files.
type Config struct {
	Scanner  ScannerConfig  `yaml:"scanner"`
	API      ApiConfig      `yaml:"api"`
	Postgres PostgresConfig `yaml:"postgres"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

type ScannerConfig struct {
	ThreadsNum int `yaml:"threads_num"`
}

type ApiConfig struct {
	Port int `yaml:"port"`
}

type PostgresConfig struct {
	Host           string `yaml:"host"`
	Port           uint64 `yaml:"port"`
	User           string `yaml:"user"`
	Pass           string `yaml:"pass"`
	Database       string `yaml:"database"`
	MigrationsPath string `yaml:"migrations_path"`
}

type MetricsConfig struct {
	IsEnabled bool `yaml:"is_enabled"`
	Port      int  `yaml:"port"`
}

func (c Config) Validate() (err error) {
	return nil
}

// LoadFile parses the given YAML file into a Config.
func LoadFile(filename string) (Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	cfg, err := Load(string(content))
	if err != nil {
		return Config{}, errors.Wrapf(err, "parsing YAML file %s", filename)
	}

	err = cfg.Validate()
	if err != nil {
		return Config{}, errors.Wrapf(err, "validate Config error: %s", filename)
	}

	return cfg, nil
}

// Load parses the YAML input s into a Config.
func Load(s string) (cfg Config, err error) {
	err = yaml.UnmarshalStrict([]byte(s), &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
