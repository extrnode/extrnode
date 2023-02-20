package postgres

import sq "github.com/Masterminds/squirrel"

type Blockchain struct {
	ID   int    `pg:"blc_id"`
	Name string `pg:"blc_name"`
}

const blockchainsTable = "blockchains"

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
