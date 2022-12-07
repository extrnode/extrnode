package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type Blockchain struct {
	ID   int    `pg:"blc_id"`
	Name string `pg:"blc_name"`
}

const blockchainsTable = "blockchains"

func (p *PgStorage) GetOrCreateBlockchain(name string) (id int, err error) {
	if name == "" {
		return id, fmt.Errorf("empty name")
	}

	query, args, err := sq.Select("blc_id").
		From(blockchainsTable).
		Where("blc_name = ?", name).ToSql()
	if err != nil {
		return id, err
	}

	m := Blockchain{}
	_, err = p.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO blockchains (blc_name)
			VALUES (?) RETURNING blc_id`

		_, err = p.db.QueryOne(&m, query, name)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}
	}

	return m.ID, nil
}

func (p *PgStorage) GetBlockchains() (res []Blockchain, err error) {
	query, args, err := sq.Select("blc_id, blc_name").
		From(blockchainsTable).ToSql()
	if err != nil {
		return res, err
	}

	_, err = p.db.Query(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}
