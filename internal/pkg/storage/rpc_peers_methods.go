package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type RpcPeerMethod struct {
	PeerID      int `pg:"prs_id"` // Peer
	RpcMethodID int `pg:"mtd_id"` // RpcMethod
}

const rpcPeersMethodsTable = "rpc.peers_methods"

func (p *PgStorage) CreateRpcPeerMethod(peerID, rpcMethodID int) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}
	if rpcMethodID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query, args, err := sq.Select("prs_id, mtd_id").
		From(rpcPeersMethodsTable).
		Where("prs_id = ? AND mtd_id = ?", peerID, rpcMethodID).ToSql()
	if err != nil {
		return err
	}

	_, err = p.db.ExecOne(query, args...)
	if err != nil && err != pg.ErrNoRows {
		return fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO rpc.peers_methods (prs_id, mtd_id)
			VALUES (?, ?)`

		_, err = p.db.Exec(query, peerID, rpcMethodID)
		if err != nil {
			return fmt.Errorf("insert: %s", err)
		}
	}

	return nil
}
