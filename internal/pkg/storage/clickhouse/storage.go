package clickhouse

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type Storage struct {
	conn *sql.DB
}

func New(dsn string) (s Storage, err error) {
	opt, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return s, err
	}
	conn := clickhouse.OpenDB(opt)
	conn.SetMaxIdleConns(5)
	conn.SetMaxOpenConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	// Test connection
	err = conn.Ping()
	if err != nil {
		return s, fmt.Errorf("connection ping error: %w", err)
	}

	return Storage{conn: conn}, nil
}

func (s *Storage) Close() error {
	return s.conn.Close()
}
