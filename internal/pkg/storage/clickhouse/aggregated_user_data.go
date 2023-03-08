package clickhouse

import (
	"fmt"
)

func (s *Storage) InsertAggregateUserData() error {
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
	_, err := s.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("exec: %s", err)
	}

	_, err = s.conn.Exec(`OPTIMIZE TABLE aggregated_user_data FINAL`)
	if err != nil {
		return fmt.Errorf("optimize: %s", err)
	}

	return nil
}
