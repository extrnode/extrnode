package clickhouse

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"extrnode-be/internal/pkg/log"
)

type Storage struct {
	conn     *sql.DB
	hostname string
}

func New(dsn, hostname string) (s *Storage, err error) {
	if dsn == "" {
		log.Logger.General.Infof("start without CH")
		return nil, nil
	}

	opt, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return s, fmt.Errorf("ParseDSN: %s", err)
	}
	conn := clickhouse.OpenDB(opt)
	conn.SetMaxIdleConns(5)
	conn.SetMaxOpenConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	// Test connection
	err = conn.Ping()
	if err != nil {
		return s, fmt.Errorf("connection ping error: %s", err)
	}

	return &Storage{conn: conn, hostname: hostname}, nil
}

func (s *Storage) Close() error {
	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}
