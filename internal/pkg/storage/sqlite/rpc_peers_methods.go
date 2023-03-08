package sqlite

import (
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type RpcPeerMethod struct {
	PeerID       int
	RpcMethodID  int
	ResponseTime int
}

const rpcPeersMethodsTable = "rpc_peers_methods"

func (s *Storage) UpsertRpcPeerMethod(peerID, rpcMethodID int, responseTime time.Duration) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}
	if rpcMethodID == 0 {
		return fmt.Errorf("empty rpcMethodID")
	}

	query := `INSERT INTO rpc_peers_methods (prs_id, mtd_id, pmd_response_time_ms)
			VALUES (?, ?, ?) ON CONFLICT DO UPDATE SET pmd_response_time_ms = ?`
	_, err := s.db.ExecContext(s.ctx, query, peerID, rpcMethodID, responseTime.Milliseconds(), responseTime.Milliseconds())
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) DeleteRpcPeerMethod(peerID int, rpcMethodID *int) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}
	if rpcMethodID != nil && *rpcMethodID == 0 {
		return fmt.Errorf("empty rpcMethodID")
	}

	request := sq.Delete(rpcPeersMethodsTable).
		Where("prs_id = ?", peerID)

	if rpcMethodID != nil {
		request = request.Where("mtd_id = ?", *rpcMethodID)
	}
	query, args, err := request.ToSql()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete: %s", err)
	}

	return nil
}
