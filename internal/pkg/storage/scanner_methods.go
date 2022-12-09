package storage

import (
	"fmt"
	"time"
)

type ScannerMethod struct {
	ID            int       `pg:"smt_id"`
	PeerID        int       `pg:"prs_id"` // Peer
	RpcMethodID   int       `pg:"mtd_id"` // RpcMethod
	Date          time.Time `pg:"smt_date"`
	TimeConnect   int       `pg:"smt_time_connect_ms"`
	TimeResponse  int       `pg:"smt_time_response_ms"`
	ResponseCode  int       `pg:"smt_response_code"`
	ResponseValid bool      `pg:"smt_response_valid"`
}

const scannerMethodsTable = "scanner.methods"

func (p *PgStorage) CreateScannerMethod(peerID, rpcMethodID int, date time.Time, timeConnect, timeResponse time.Duration, responseCode int, responseValid bool) error {
	if peerID == 0 {
		return fmt.Errorf("empty PeerID")
	}
	if rpcMethodID == 0 {
		return fmt.Errorf("empty rpcMethodID")
	}

	query := `INSERT INTO scanner.methods (prs_id, mtd_id, smt_date, smt_time_connect_ms, smt_time_response_ms, smt_response_code, smt_response_valid)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := p.db.Exec(query, peerID, rpcMethodID, date.UTC(), timeConnect.Milliseconds(), timeResponse.Milliseconds(), responseCode, responseValid)
	if err != nil {
		return err
	}

	return nil
}
