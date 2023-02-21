package config

import (
	"errors"
	"fmt"
)

func (c Config) validate() error {
	if err := c.SApi.validate(); err != nil {
		return fmt.Errorf("scanner_api: %s", err)
	}
	if err := c.UApi.validate(); err != nil {
		return fmt.Errorf("user_api: %s", err)
	}
	if err := c.Proxy.validate(); err != nil {
		return fmt.Errorf("proxy: %s", err)
	}
	if err := c.SL.validate(); err != nil {
		return fmt.Errorf("sqlite: %s", err)
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

func (e ScannerApiConfig) validate() error {
	if e.Port == 0 {
		return errors.New("invalid port")
	}

	return nil
}

func (u UserApiConfig) validate() error {
	if u.Port == 0 {
		return errors.New("invalid port")
	}
	if u.FirebaseFilePath == "" {
		return errors.New("invalid firebase file path")
	}

	return nil
}

func (p ProxyConfig) validate() error {
	if p.Port == 0 {
		return errors.New("invalid port")
	}
	for _, h := range p.FailoverEndpoints {
		if h.Url == "" {
			return errors.New("invalid failover endpoints")
		}
	}

	return nil
}

func (p SQLiteConfig) validate() error {
	if p.DBPath == "" {
		return fmt.Errorf("invalid DBPath")
	}
	if p.MigrationsPath == "" {
		return fmt.Errorf("invalid MigrationsPath")
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
	//if c.DSN == "" {
	//	return fmt.Errorf("invalid dsn")
	//}

	return nil
}

func (s ScannerConfig) validate() error {
	if s.ThreadsNum <= 0 {
		return fmt.Errorf("invalid ThreadsNum: %d", s.ThreadsNum)
	}
	if s.Hostname == "" {
		return fmt.Errorf("empty Hostname")
	}

	return nil
}
