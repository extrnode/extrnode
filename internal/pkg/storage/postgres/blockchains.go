package postgres

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

func (p *Storage) GetOrCreateBlockchain(name string) (id int, err error) {
	if name == "" {
		return id, fmt.Errorf("empty name")
	}

	query, args, err := sq.Select("blc_id").
		From(blockchainsTable).
		Where("blc_name = ?", name).ToSql()
	if err != nil {
		return id, err
	}

	s, err := p.BeginTx()
	if err != nil {
		return id, fmt.Errorf("beginTx: %s", err)
	}
	defer s.Rollback()

	m := Blockchain{}
	_, err = s.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO blockchains (blc_name)
			VALUES (?) RETURNING blc_id`

		_, err = s.db.QueryOne(&m, query, name)
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

func (p *Storage) GetBlockchainsMap() (res map[string]int, err error) {
	query, args, err := sq.Select("blc_id, blc_name").
		From(blockchainsTable).ToSql()
	if err != nil {
		return res, err
	}

	var ms []Blockchain
	_, err = p.db.Query(&ms, query, args...)
	if err != nil {
		return res, err
	}

	res = make(map[string]int)
	for _, m := range ms {
		res[m.Name] = m.ID
	}

	return res, nil
}

func (p *Storage) GetBlockchainByName(name string) (res Blockchain, err error) {
	query, args, err := sq.Select("blc_id, blc_name").
		From(blockchainsTable).
		Where("blc_name = ?", name).
		ToSql()
	if err != nil {
		return res, err
	}

	_, err = p.db.QueryOne(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}
