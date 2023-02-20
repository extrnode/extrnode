package sqlite

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type GeoCountry struct {
	ID     int
	Alpha2 string
	Alpha3 string
	Name   string
}

const geoCountriesTable = "geo_countries"

func (s *Storage) GetOrCreateGeoCountry(alpha2, alpha3, name string) (id int, err error) {
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
		query = `INSERT INTO geo_countries (cnt_alpha2, cnt_alpha3, cnt_name)
			VALUES (?, ?, ?) RETURNING cnt_id`

		err = tx.QueryRowContext(s.ctx, query, alpha2, alpha3, name).Scan(&id)
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
