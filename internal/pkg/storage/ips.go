package storage

import (
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type IP struct {
	ID        int    `pg:"ip_id"`
	NetworkID int    `pg:"ntw_id"` // GeoNetwork
	Address   net.IP `pg:"ip_addr"`
}

const ipsTable = "ips"

func (p *PgStorage) GetOrCreateIP(networkID int, address net.IP) (id int, err error) {
	if networkID == 0 {
		return id, fmt.Errorf("empty networkID")
	}

	query, args, err := sq.Select("ip_id").
		From(ipsTable).
		Where("ip_addr = ?", address).ToSql()
	if err != nil {
		return id, err
	}

	m := IP{}
	s, err := p.BeginTx()
	if err != nil {
		return id, fmt.Errorf("beginTx: %s", err)
	}
	defer s.Rollback()

	_, err = s.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO ips (ntw_id, ip_addr)
			VALUES (?, ?) RETURNING ip_id`

		_, err = s.db.QueryOne(&m, query, networkID, address)
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
