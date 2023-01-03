package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type (
	RpcMethod struct {
		ID           int    `pg:"mtd_id"`
		BlockchainID int    `pg:"blc_id"` // Blockchain
		Name         string `pg:"mtd_name"`
	}

	RpcMethodsMap map[string]int
)

const rpcMethodsTable = "rpc.methods"

func (p *PgStorage) GetOrCreateRpcMethod(blockchainID int, name string) (id int, err error) {
	if blockchainID == 0 {
		return id, fmt.Errorf("empty blockchainID")
	}
	if len(name) == 0 {
		return id, fmt.Errorf("empty name")
	}

	query, args, err := sq.Select("mtd_id").
		From(rpcMethodsTable).
		Where("blc_id = ? AND mtd_name = ?", blockchainID, name).ToSql()
	if err != nil {
		return id, err
	}

	m := RpcMethod{}
	s, err := p.BeginTx()
	if err != nil {
		return id, fmt.Errorf("beginTx: %s", err)
	}
	defer s.Rollback()

	_, err = s.db.QueryOne(m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO rpc.methods (blc_id, mtd_name)
			VALUES (?, ?) RETURNING mtd_id`

		_, err = s.db.QueryOne(m, query, blockchainID, name)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}
	}

	err = s.Commit()
	if err != nil {
		return id, fmt.Errorf("commit: %s", err)
	}

	return m.ID, nil
}

func (p *PgStorage) GetRpcMethodsMapByBlockchainID(blockchainID int) (res RpcMethodsMap, err error) {
	if blockchainID == 0 {
		return nil, fmt.Errorf("empty blockchainID")
	}

	query, args, err := sq.Select("mtd_id, mtd_name").
		From(rpcMethodsTable).
		Where("blc_id = ?", blockchainID).
		ToSql()
	if err != nil {
		return res, err
	}

	var methods []RpcMethod
	_, err = p.db.Query(&methods, query, args...)
	if err != nil {
		return res, err
	}
	res = make(RpcMethodsMap, len(methods))
	for _, m := range methods {
		res[m.Name] = m.ID
	}

	return res, nil
}
