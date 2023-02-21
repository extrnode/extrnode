package config_types

import (
	"errors"
	"fmt"
)

func (e ScannerApiConfig) Validate() error {
	if e.Port == 0 {
		return errors.New("invalid port")
	}

	return nil
}

func (u UserApiConfig) Validate() error {
	if u.Port == 0 {
		return errors.New("invalid port")
	}
	if u.FirebaseFilePath == "" {
		return errors.New("invalid firebase file path")
	}

	return nil
}

func (p ProxyConfig) Validate() error {
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

func (p SQLiteConfig) Validate() error {
	if p.DBPath == "" {
		return fmt.Errorf("invalid DBPath")
	}
	if p.MigrationsPath == "" {
		return fmt.Errorf("invalid MigrationsPath")
	}

	return nil
}

func (p PostgresConfig) Validate() error {
	if p.Port == 0 {
		return fmt.Errorf("invalid port")
	}

	return nil
}

func (s ScannerConfig) Validate() error {
	if s.ThreadsNum <= 0 {
		return fmt.Errorf("invalid ThreadsNum: %d", s.ThreadsNum)
	}
	if s.Hostname == "" {
		return fmt.Errorf("empty Hostname")
	}

	return nil
}
