package clickhouse

import (
	"database/sql"
	"fmt"

	"extrnode-be/internal/pkg/log"
)

func (s *Storage) InsertAggregateUserData() error {
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

	query := `INSERT INTO aggregated_user_data (user_uuid, rpc_method, total_req, success_req, http_err, rpc_err, day) 
	SELECT  user_uuid,
			rpc_method,
			count(rpc_method) as c,
			count(if(rpc_error_code == '' AND status == 200 AND rpc_method != '', true, null)),
			count(if(status != 200, true, null)),
			count(if(rpc_error_code != '', true, null)),
			toDate(timestamp) as day
	FROM stats
	WHERE day < toDate(now(), 'Etc/UTC')
	GROUP BY rpc_method, user_uuid, day`

	_, err = tx.Exec(query)
	if err != nil {
		return fmt.Errorf("exec statement error: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx commit error: %s", err)
	}

	_, err = s.conn.Exec(`OPTIMIZE TABLE aggregated_user_data FINAL`)
	if err != nil {
		return fmt.Errorf("optimize: %s", err)
	}

	return nil
}
