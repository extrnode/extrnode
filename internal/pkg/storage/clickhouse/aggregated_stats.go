package clickhouse

import (
	"fmt"
)

func (s *Storage) InsertAggregateAnalysisStats() error {
	query := `INSERT INTO aggregated_analysis_data (rpc_method, rpc_request_data, execution_time_ms, response_time_ms, total_req, day) 
        SELECT rpc_method,
            rpc_request_data,
            avg(execution_time_ms),
            avg(response_time_ms),
            count(rpc_method) as c,
            toDate(timestamp) as day
		FROM stats
		WHERE day < toDate(now(), 'Etc/UTC')
		GROUP BY rpc_method, rpc_request_data, day
		ORDER BY rpc_method, c desc
		LIMIT 100 BY rpc_method, day`
	_, err := s.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("exec: %s", err)
	}

	_, err = s.conn.Exec(`OPTIMIZE TABLE aggregated_analysis_data FINAL`)
	if err != nil {
		return fmt.Errorf("optimize: %s", err)
	}

	return nil
}
