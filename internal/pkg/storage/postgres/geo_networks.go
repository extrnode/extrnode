package postgres

import (
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type GeoNetwork struct {
	ID        int       `pg:"ntw_id"`
	CountryID int       `pg:"cnt_id"` // GeoCountry
	Mask      net.IPNet `pg:"ntw_mask"`
	As        int       `pg:"ntw_as"`
	Name      string    `pg:"ntw_name"`
}

const geoNetworksTable = "geo.networks"

func (p *Storage) GetOrCreateGeoNetwork(countryID int, mask net.IPNet, as int, name string) (id int, err error) {
	if countryID == 0 {
		return id, fmt.Errorf("empty countryID")
	}

	query, args, err := sq.Select("ntw_id").
		From(geoNetworksTable).
		Where("cnt_id = ? AND ntw_mask = ?", countryID, mask).ToSql()
	if err != nil {
		return id, err
	}

	s, err := p.BeginTx()
	if err != nil {
		return id, fmt.Errorf("beginTx: %s", err)
	}
	defer s.Rollback()

	m := GeoNetwork{}
	_, err = s.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO geo.networks (cnt_id, ntw_mask, ntw_as, ntw_name)
			VALUES (?, ?, ?, ?) RETURNING ntw_id`

		_, err = s.db.QueryOne(&m, query, countryID, mask, as, name)
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

func (p *Storage) GetGeoNetworks() (res []GeoNetwork, err error) {
	query, args, err := sq.Select("ntw_id, cnt_id, ntw_mask, ntw_as, ntw_name").
		From(geoNetworksTable).ToSql()
	if err != nil {
		return res, err
	}

	_, err = p.db.Query(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}
