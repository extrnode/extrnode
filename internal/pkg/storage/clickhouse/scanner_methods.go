package clickhouse

import (
	"database/sql"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/log"
)

type ScannerMethod struct {
	ServerId       string
	Time           time.Time
	Peer           string
	Method         string
	TimeConnectMs  int64
	TimeResponseMs int64
	ResponseCode   uint16
	ResponseValid  bool
}

func (s *Storage) BatchInsertScannerMethods(sms []ScannerMethod) error {
	if len(sms) == 0 {
		return nil
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return fmt.Errorf("tx begin error: %s", err)
	}

	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			log.Logger.General.Errorf("tx rollback error: %s", err)
		}
	}()

	stmt, err := tx.Prepare(`INSERT INTO scanner_methods (
        server_id,
        time,
        peer,
        method,
        time_connect_ms,
        time_response_ms,
        response_code,
        response_valid
	)`)
	if err != nil {
		return fmt.Errorf("prepare statement error: %s", err)
	}

	for _, sm := range sms {
		_, err = stmt.Exec(
			sm.ServerId,
			sm.Time,
			sm.Peer,
			sm.Method,
			sm.TimeConnectMs,
			sm.TimeResponseMs,
			sm.ResponseCode,
			sm.ResponseValid,
		)
		if err != nil {
			return fmt.Errorf("exec statement error: %s", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx commit error: %s", err)
	}

	return nil
}
