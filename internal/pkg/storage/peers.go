package storage

import (
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"

	"extrnode-be/internal/models"
	"extrnode-be/internal/pkg/log"
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

func (p *PgStorage) GetEndpoints(blockchain string, limit int, isRpc *bool, asnCountries, versions, supportedMethods []string) (res []models.Endpoint, err error) {
	q := sq.Select(`CONCAT(ip_addr, ':', prs_port)  AS endpoint,
		   prs_version 										  AS version,
		   prs_is_rpc 										  AS node_type,
		   json_agg(rpc.methods.mtd_name)                     AS supported_methods,
		   json_build_object('network', ntw_mask, 'isp', ntw_name, 'country',
									  json_build_object('alpha2', cnt_alpha2, 'alpha3', cnt_alpha3, 'name',
														cnt_name))  AS asn_info,
		   COALESCE(SUM(scanner.methods.smt_time_response_ms), 0)   AS scan_time`).
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING (ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING (ntw_id)", geoNetworksTable)).
		LeftJoin(fmt.Sprintf("%s USING (cnt_id)", geoCountriesTable)).
		LeftJoin(fmt.Sprintf("%s USING (prs_id)", rpcPeersMethodsTable)).
		LeftJoin(fmt.Sprintf("%s USING (mtd_id)", rpcMethodsTable)).
		LeftJoin(fmt.Sprintf(`%s ON peers.prs_id = scanner.methods.prs_id AND scanner.methods.smt_date =
                                                                                (SELECT MAX(smt_date)
                                                                                 FROM scanner.methods
                                                                                 WHERE peers.prs_id = methods.prs_id)`, scannerMethodsTable)).
		Where("prs_is_alive IS TRUE AND peers.blc_id = (SELECT blc_id FROM blockchains WHERE blc_name = ?)", blockchain).
		GroupBy("peers.prs_id, ip_addr, ntw_mask, ntw_name, cnt_alpha2, cnt_alpha3, cnt_name")
	if isRpc != nil {
		q = q.Where("prs_is_rpc = ?", *isRpc)
	}
	if len(asnCountries) != 0 {
		q = q.Where(sq.Eq{"cnt_alpha2": asnCountries})
	}
	if len(supportedMethods) != 0 {
		q = q.Where(sq.Eq{"rpc.methods.mtd_name": supportedMethods})
	}
	if len(versions) != 0 {
		q = q.Where(LikeAny{"prs_version": versions})
	}
	if limit != 0 {
		q = q.Limit(uint64(limit))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return res, err
	}

	log.Logger.Api.Debugf("%s", query)
	_, err = p.db.Query(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}
