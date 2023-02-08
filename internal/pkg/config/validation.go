package config

import (
	"errors"
	"fmt"
)

func (c Config) validate() error {
	if err := c.API.validate(); err != nil {
		return fmt.Errorf("api: %s", err)
	}
	if err := c.PG.validate(); err != nil {
		return fmt.Errorf("postgres: %s", err)
	}
	if err := c.CH.validate(); err != nil {
		return fmt.Errorf("clickhouse: %s", err)
	}
	if err := c.Scanner.validate(); err != nil {
		return fmt.Errorf("scanner: %s", err)
	}

	return nil
}

func (a ApiConfig) validate() error {
	if a.Port == 0 {
		return errors.New("invalid port")
	}
	for _, h := range a.FailoverEndpoints {
		if h.Url == "" {
			return errors.New("invalid failover endpoints")
		}
	}

	return nil
}

func (p PostgresConfig) validate() error {
	if p.Port == 0 {
		return fmt.Errorf("invalid port")
	}

	return nil
}

func (c ClickhouseConfig) validate() error {
	if c.DSN == "" {
		return fmt.Errorf("invalid dsn")
	}

	return nil
}

func (s ScannerConfig) validate() error {
	if s.ThreadsNum <= 0 {
		return fmt.Errorf("invalid ThreadsNum: %d", s.ThreadsNum)
	}

	return nil
}
