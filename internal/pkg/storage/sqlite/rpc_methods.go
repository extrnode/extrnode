package sqlite

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type (
	RpcMethod struct {
		ID           int
		BlockchainID int
		Name         string
	}
)

const rpcMethodsTable = "rpc_methods"

func (s *Storage) GetOrCreateRpcMethod(blockchainID int, name string) (id int, err error) {
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

	tx, err := s.db.BeginTx(s.ctx, nil)
	if err != nil {
		return id, fmt.Errorf("tx begin error: %s", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(s.ctx, query, args...).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == sql.ErrNoRows {
		query = `INSERT INTO rpc_methods (blc_id, mtd_name)
			VALUES (?, ?) RETURNING mtd_id`

		err = tx.QueryRowContext(s.ctx, query, blockchainID, name).Scan(&id)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return id, fmt.Errorf("tx commit error: %s", err)
	}

	return
}

func (s *Storage) GetRpcMethodsMapByBlockchainID(blockchainID int) (res map[string]int, err error) {
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

	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	res = make(map[string]int)
	for rows.Next() {
		var method RpcMethod
		if err = rows.Scan(&method.ID, &method.Name); err != nil {
			return res, err
		}
		res[method.Name] = method.ID
	}

	return res, nil
}
