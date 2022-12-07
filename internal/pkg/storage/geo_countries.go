package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type GeoCountry struct {
	ID     int    `pg:"cnt_id"`
	Alpha2 string `pg:"cnt_alpha2"`
	Alpha3 string `pg:"cnt_alpha3"`
	Name   string `pg:"cnt_name"`
}

const geoCountriesTable = "geo.countries"

func (p *PgStorage) GetOrCreateGeoCountry(alpha2, alpha3, name string) (id int, err error) {
	if alpha2 == "" {
		return id, fmt.Errorf("empty alpha2")
	}
	if alpha3 == "" {
		return id, fmt.Errorf("empty alpha3")
	}

	query, args, err := sq.Select("cnt_id").
		From(geoCountriesTable).
		Where("cnt_alpha2 = ? AND cnt_alpha3 = ?", alpha2, alpha3).ToSql()
	if err != nil {
		return id, err
	}

	m := GeoCountry{}
	_, err = p.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO geo.countries (cnt_alpha2, cnt_alpha3, cnt_name)
			VALUES (?, ?, ?) RETURNING cnt_id`

		_, err = p.db.QueryOne(&m, query, alpha2, alpha3, name)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}
	}

	return m.ID, nil
}
