package config

import "fmt"

func (c Config) validate() error {
	if err := c.API.validate(); err != nil {
		return fmt.Errorf("api: %s", err)
	}
	if err := c.PG.validate(); err != nil {
		return fmt.Errorf("postgres: %s", err)
	}
	if err := c.Scanner.validate(); err != nil {
		return fmt.Errorf("scanner: %s", err)
	}

	return nil
}

func (a ApiConfig) validate() error {
	return nil
}

func (p PostgresConfig) validate() error {
	if p.Port == 0 {
		return fmt.Errorf("invalid port")
	}

	return nil
}

func (s ScannerConfig) validate() error {
	if s.ThreadsNum <= 0 {
		return fmt.Errorf("invalid ThreadsNum: %d", s.ThreadsNum)
	}

	return nil
}
