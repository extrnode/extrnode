package storage

import (
	"fmt"
	"net"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"

	"extrnode-be/internal/models"
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
		IsMainNet    bool   `pg:"prs_is_main_net"`
		NodePubkey   string `pg:"prs_node_pubkey"`
		IsValidator  bool   `pg:"prs_is_validator"`
	}

	PeerWithIp struct {
		Address net.IP `pg:"ip_addr"`
		Peer
	}
	PeerWithIpAndBlockchain struct {
		Peer
		Address        net.IP `pg:"ip_addr"`
		BlockchainName string `pg:"blc_name"`
	}
)

const peersTable = "peers"

func (p *PgStorage) GetOrCreatePeer(blockchainID, ipID, port int, version string, isRpc, isAlive, isSSL, isMainNet, isValidator bool, nodePubkey string) (id int, err error) {
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
		query = `INSERT INTO peers (blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, prs_is_main_net, prs_is_validator, prs_node_pubkey)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING prs_id`

		_, err = s.db.QueryOne(&m, query, blockchainID, ipID, port, version, isRpc, isAlive, isSSL, isMainNet, isValidator, nodePubkey)
		if err != nil {
			return id, fmt.Errorf("insert: %s", err)
		}

		err = s.Commit()
		if err != nil {
			return id, fmt.Errorf("commit: %s", err)
		}
	}

	return m.ID, nil
}

func (p *PgStorage) UpdatePeerByID(peerID int, isRpc, isAlive, isSSL, isMainNet, isValidator bool) (err error) {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `UPDATE peers SET prs_is_rpc = ?, prs_is_alive = ?, prs_is_ssl = ?, prs_is_main_net = ?, prs_is_validator = ?
			WHERE prs_id = ?`
	_, err = p.db.Exec(query, isRpc, isAlive, isSSL, isMainNet, isValidator, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (p *PgStorage) UpdatePeerVersionAndNodePubkey(peerID int, version, nodePubkey string) (err error) {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `UPDATE peers SET prs_version = ?, prs_node_pubkey = ?
			WHERE prs_id = ?`
	_, err = p.db.Exec(query, version, nodePubkey, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (p *PgStorage) GetEndpoints(blockchainID, limit int, isRpc, isValidator *bool, asnCountries, versions, supportedMethods []string) (res []models.Endpoint, err error) {
	if blockchainID == 0 {
		return nil, fmt.Errorf("empty blockchainID")
	}
	for i := range asnCountries {
		asnCountries[i] = strings.ToUpper(asnCountries[i])
	}

	q := sq.Select(`CONCAT(ip_addr, ':', prs_port)  AS endpoint,
		   prs_version 										  AS version,
		   prs_is_rpc 										  AS is_rpc,
		   prs_is_validator 								  AS is_validator,
		   json_agg(json_build_object('name', rpc.methods.mtd_name, 'response_time', rpc.peers_methods.pmd_response_time_ms)) AS supported_methods,
		   json_build_object('network', ntw_mask, 'isp', ntw_name, 'ntw_as', ntw_as, 'country',
									  json_build_object('alpha2', cnt_alpha2, 'alpha3', cnt_alpha3, 'name', cnt_name)) AS asn_info`).
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING (ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING (ntw_id)", geoNetworksTable)).
		LeftJoin(fmt.Sprintf("%s USING (cnt_id)", geoCountriesTable)).
		LeftJoin(fmt.Sprintf("%s USING (prs_id)", rpcPeersMethodsTable)).
		LeftJoin(fmt.Sprintf("%s USING (mtd_id)", rpcMethodsTable)).
		Where("prs_is_alive IS TRUE AND prs_is_main_net IS TRUE AND peers.blc_id = ?", blockchainID).
		GroupBy("peers.prs_id, ip_addr, ntw_mask, ntw_name, ntw_as, cnt_alpha2, cnt_alpha3, cnt_name")
	if isRpc != nil {
		q = q.Where("prs_is_rpc = ?", *isRpc)
	}
	if isValidator != nil {
		q = q.Where("prs_is_validator = ?", *isValidator)
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

	_, err = p.db.Query(&res, query, args...)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (p *PgStorage) GetPeers() (res []PeerWithIpAndBlockchain, err error) {
	query, args, err := sq.Select("prs_id, blc_id, blc_name, ip_id, ip_addr, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, prs_is_main_net, prs_node_pubkey, prs_is_validator").
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING(ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING(blc_id)", blockchainsTable)).
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

func (p *PgStorage) GetExistentPeers(blockchainID int, ips []net.IP) (res map[string]map[int]PeerWithIp, err error) {
	if len(ips) == 0 {
		return res, nil
	}
	if blockchainID == 0 {
		return res, fmt.Errorf("empty blockchainID")
	}

	query, args, err := sq.Select("prs_id, prs_port, prs_version, prs_node_pubkey, ip_addr").
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING(ip_id)", ipsTable)).
		Where("blc_id = ?", blockchainID).
		Where(sq.Eq{"ip_addr": ips}).
		ToSql()
	if err != nil {
		return res, err
	}

	var peers []PeerWithIp
	_, err = p.db.Query(&peers, query, args...)
	if err != nil {
		return res, err
	}

	res = make(map[string]map[int]PeerWithIp, len(peers)) // ip/port
	for _, peer := range peers {
		if _, ok := res[peer.Address.String()]; !ok {
			res[peer.Address.String()] = make(map[int]PeerWithIp)
		}

		res[peer.Address.String()][peer.Port] = peer
	}

	return res, nil
}

func (p *PgStorage) GetStats() (res models.Stat, err error) {
	q := `SELECT COUNT(*)                                                   					AS total,
		   SUM(CASE WHEN prs_is_alive IS true THEN 1 ELSE 0 END)     							AS alive,
		   SUM(CASE WHEN prs_is_rpc IS true THEN 1 ELSE 0 END)       							AS rpc,
		   SUM(CASE WHEN prs_is_alive IS true AND prs_is_validator IS true THEN 1 ELSE 0 END) 	AS validator
		FROM peers WHERE prs_is_main_net IS TRUE`

	_, err = p.db.QueryOne(&res, q)
	if err != nil {
		return res, err
	}

	return res, nil
}
