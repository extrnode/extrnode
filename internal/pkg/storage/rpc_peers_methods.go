package storage

import (
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type RpcPeerMethod struct {
	PeerID       int `pg:"prs_id"` // Peer
	RpcMethodID  int `pg:"mtd_id"` // RpcMethod
	ResponseTime int `pg:"pmd_response_time_ms"`
}

const rpcPeersMethodsTable = "rpc.peers_methods"

func (p *PgStorage) UpsertRpcPeerMethod(peerID, rpcMethodID int, responseTime time.Duration) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}
	if rpcMethodID == 0 {
		return fmt.Errorf("empty rpcMethodID")
	}

	query := `INSERT INTO rpc.peers_methods (prs_id, mtd_id, pmd_response_time_ms)
			VALUES (?, ?, ?) ON CONFLICT ON CONSTRAINT peers_methods_pk DO UPDATE SET pmd_response_time_ms = ?`
	_, err := p.db.Exec(query, peerID, rpcMethodID, responseTime.Milliseconds(), responseTime.Milliseconds())
	if err != nil {
		return err
	}

	return nil
}

func (p *PgStorage) DeleteRpcPeerMethod(peerID, rpcMethodID int) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}
	if rpcMethodID == 0 {
		return fmt.Errorf("empty rpcMethodID")
	}

	query, args, err := sq.Delete(rpcPeersMethodsTable).
		Where("prs_id = ? AND mtd_id = ?", peerID, rpcMethodID).ToSql()
	if err != nil {
		return err
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("delete: %s", err)
	}

	return nil
}
