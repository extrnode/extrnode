package clickhouse

import (
	"database/sql"
	"fmt"

	"extrnode-be/internal/pkg/log"
)

type Stat struct {
	UserUUID      string
	RequestID     string
	Status        uint16
	ExecutionTime int64
	Endpoint      string
	Attempts      uint8
	ResponseTime  int64
	RpcErrorCode  string
	UserAgent     string
	RpcMethod     string
	// for sendTransaction - should be the programID
	// for getSignaturesForAddress - address
	// for getTokenAccountsByOwner - owner
	// for getAccountInfo - account
	// for getProgramAccounts - programID
	RpcRequestData string
}

func (s *Storage) BatchInsertStats(stats []Stat) error {
	if len(stats) == 0 {
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

	stmt, err := tx.Prepare(`INSERT INTO stats (
		user_uuid,
        request_id,
        status,
        execution_time_ms,
        endpoint,
        attempts,
        response_time_ms,
        rpc_error_code,
        user_agent,
        rpc_method,
        rpc_request_data
	)`)
	if err != nil {
		return fmt.Errorf("prepare statement error: %s", err)
	}

	for _, s := range stats {
		_, err = stmt.Exec(
			s.UserUUID,
			s.RequestID,
			s.Status,
			s.ExecutionTime,
			s.Endpoint,
			s.Attempts,
			s.ResponseTime,
			s.RpcErrorCode,
			s.UserAgent,
			s.RpcMethod,
			s.RpcRequestData,
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
