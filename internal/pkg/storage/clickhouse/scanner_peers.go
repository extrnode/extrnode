package clickhouse

import (
	"database/sql"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/log"
)

type ScannerPeer struct {
	ServerId      string
	Time          time.Time
	Peer          string
	TimeConnectMs int64
	IsAlive       bool
}

func (s *Storage) BatchInsertScannerPeers(sps []ScannerPeer) error {
	if len(sps) == 0 {
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

	stmt, err := tx.Prepare(`INSERT INTO scanner_peers (
        server_id,
        time,
        peer,
        time_connect_ms,
        is_alive
	)`)
	if err != nil {
		return fmt.Errorf("prepare statement error: %s", err)
	}

	for _, sm := range sps {
		_, err = stmt.Exec(
			s.hostname,
			sm.Time,
			sm.Peer,
			sm.TimeConnectMs,
			sm.IsAlive,
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
