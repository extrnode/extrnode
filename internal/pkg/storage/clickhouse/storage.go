package clickhouse

import (
	"database/sql"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"extrnode-be/internal/pkg/log"
)

type Storage struct {
	conn *sql.DB
}

func New(dsn string) (s *Storage, err error) {
	opt, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		log.Logger.Proxy.Errorf("CH.New: %s", err)
		return s, nil
	}
	conn := clickhouse.OpenDB(opt)
	conn.SetMaxIdleConns(5)
	conn.SetMaxOpenConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	// Test connection
	err = conn.Ping()
	if err != nil {
		log.Logger.Proxy.Errorf("CH.New: connection ping error: %s", err)
		return s, nil
	}

	return &Storage{conn: conn}, nil
}

func (s *Storage) Close() error {
	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}
