package sqlite

import (
	"database/sql"
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
)

type IP struct {
	ID        int
	NetworkID int
	Address   net.IP
}

const ipsTable = "ips"

func (s *Storage) GetOrCreateIP(networkID int, address net.IP) (id int, err error) {
	if networkID == 0 {
		return id, fmt.Errorf("empty networkID")
	}

	query, args, err := sq.Select("ip_id").
		From(ipsTable).
		Where("ip_addr = ?", address.String()).ToSql()
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
		query = `INSERT INTO ips (ntw_id, ip_addr)
			VALUES (?, ?) RETURNING ip_id`

		err = tx.QueryRowContext(s.ctx, query, networkID, address.String()).Scan(&id)
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
