package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
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

// LoadFile parses the given YAML file into a Config.
func LoadFile(filename string) (c Config, err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return c, err
	}
	cfg, err := Load(content)
	if err != nil {
		return c, fmt.Errorf("parsing YAML file %s: %s", filename, err)
	}

	err = cfg.validate()
	if err != nil {
		return c, fmt.Errorf("validate %s: %s", filename, err)
	}

	return cfg, nil
}

// Load parses the YAML input s into a Config.
func Load(s []byte) (cfg Config, err error) {
	d := yaml.NewDecoder(bytes.NewBuffer(s))
	d.KnownFields(true)
	err = d.Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
