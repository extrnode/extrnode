package sqlite

import (
	"database/sql"
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
)

type GeoNetwork struct {
	ID        int
	CountryID int
	Mask      net.IPNet
	As        int
	Name      string
}

const geoNetworksTable = "geo_networks"

func (s *Storage) GetOrCreateGeoNetwork(countryID int, mask net.IPNet, as int, name string) (id int, err error) {
	if countryID == 0 {
		return id, fmt.Errorf("empty countryID")
	}

	query, args, err := sq.Select("ntw_id").
		From(geoNetworksTable).
		Where("cnt_id = ? AND ntw_mask = ?", countryID, mask.String()).ToSql()
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
		query = `INSERT INTO geo_networks (cnt_id, ntw_mask, ntw_as, ntw_name)
			VALUES (?, ?, ?, ?) RETURNING ntw_id`

		err = tx.QueryRowContext(s.ctx, query, countryID, mask.String(), as, name).Scan(&id)
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
