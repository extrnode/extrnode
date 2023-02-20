package sqlite

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type Blockchain struct {
	ID   int
	Name string
}

const blockchainsTable = "blockchains"

func (s *Storage) GetOrCreateBlockchain(name string) (id int, err error) {
	if name == "" {
		return id, fmt.Errorf("empty name")
	}

	tx, err := s.db.BeginTx(s.ctx, nil)
	if err != nil {
		return id, fmt.Errorf("tx begin error: %s", err)
	}
	defer tx.Rollback()

	query, args, err := sq.Select("blc_id").
		From(blockchainsTable).
		Where("blc_name = ?", name).ToSql()
	if err != nil {
		return id, err
	}

	err = tx.QueryRowContext(s.ctx, query, args...).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == sql.ErrNoRows {
		query = `INSERT INTO blockchains (blc_name)
			VALUES (?) RETURNING blc_id`

		err = tx.QueryRowContext(s.ctx, query, name).Scan(&id)
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

func (s *Storage) GetBlockchainsMap() (res map[string]int, err error) {
	query, args, err := sq.Select("blc_id, blc_name").
		From(blockchainsTable).ToSql()
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
		var blockchain Blockchain
		if err = rows.Scan(&blockchain.ID, &blockchain.Name); err != nil {
			return res, err
		}
		res[blockchain.Name] = blockchain.ID
	}

	return res, nil
}

func (s *Storage) GetBlockchainByName(name string) (res Blockchain, err error) {
	query, args, err := sq.Select("blc_id, blc_name").
		From(blockchainsTable).
		Where("blc_name = ?", name).
		ToSql()
	if err != nil {
		return res, err
	}

	err = s.db.QueryRowContext(s.ctx, query, args...).Scan(&res.ID, &res.Name)
	if err != nil {
		return res, err
	}

	return res, nil
}
