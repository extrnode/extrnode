package storage

import (
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
)

type (
	Peer struct {
		ID           int    `pg:"prs_id"`
		BlockchainID int    `pg:"blc_id"` // Blockchain
		IpID         int    `pg:"ip_id"`  // IP
		Port         int    `pg:"prs_port"`
		Version      string `pg:"prs_version"`
		IsRpc        bool   `pg:"prs_is_rpc"`
		IsAlive      bool   `pg:"prs_is_alive"`
		IsSSL        bool   `pg:"prs_is_ssl"`
	}

	PeerWithIp struct {
		Peer
		Address net.IP `pg:"ip_addr"`
	}
)

const peersTable = "peers"

func (p *PgStorage) GetOrCreatePeer(blockchainID, ipID, port int, version string, isRpc, isAlive, isSSL bool) (id int, err error) {
	if blockchainID == 0 {
		return id, fmt.Errorf("empty blockchainID")
	}
	if ipID == 0 {
		return id, fmt.Errorf("empty ipID")
	}
	if port == 0 {
		return id, fmt.Errorf("empty port")
	}

	query, args, err := sq.Select("prs_id").
		From(peersTable).
		Where("blc_id = ? AND ip_id = ? AND prs_port = ?", blockchainID, ipID, port).ToSql()
	if err != nil {
		return id, err
	}

	m := Peer{}
	_, err = p.db.QueryOne(&m, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return id, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		query = `INSERT INTO peers (blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl)
			VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING prs_id`

		_, err = p.db.QueryOne(&m, query, blockchainID, ipID, port, version, isRpc, isAlive, isSSL)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}
	}

	return m.ID, nil
}

func (p *PgStorage) GetPeersByBlockchainID(blockchainID int) (res []PeerWithIp, err error) {
	if blockchainID == 0 {
		return nil, fmt.Errorf("empty blockchainID")
	}

	query, args, err := sq.Select("prs_id, blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, ip_addr").
		From(peersTable).
		Where("blc_id = ?", blockchainID).
		Join(fmt.Sprintf("%s USING(ip_id)", ipsTable)).
		ToSql()
	if err != nil {
		return res, err
	}

	_, err = p.db.Query(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}
