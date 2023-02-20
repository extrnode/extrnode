package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"extrnode-be/internal/models"
	"extrnode-be/internal/pkg/log"
)

type (
	Peer struct {
		ID           int
		BlockchainID int
		IpID         int
		Port         int
		Version      string
		IsRpc        bool
		IsAlive      bool
		IsSSL        bool
		IsMainNet    bool
		NodePubkey   string
		IsValidator  bool
		IsOutdated   bool
	}

	PeerWithIp struct {
		Address net.IP
		Peer
	}
	PeerWithIpAndBlockchain struct {
		Peer
		Address        net.IP
		BlockchainName string
	}
)

const peersTable = "peers"

func (s *Storage) GetOrCreatePeer(blockchainID, ipID, port int, version string, isRpc, isAlive, isSSL, isMainNet, isValidator bool, nodePubkey string) (id int, err error) {
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
		query = `INSERT INTO peers (blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, prs_is_main_net, prs_is_validator, prs_node_pubkey)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING prs_id`

		err = tx.QueryRowContext(s.ctx, query, blockchainID, ipID, port, version, isRpc, isAlive, isSSL, isMainNet, isValidator, nodePubkey).Scan(&id)
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

func (s *Storage) UpdatePeerByID(peerID int, isRpc, isAlive, isSSL, isMainNet, isValidator bool, version string) (err error) {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `UPDATE peers SET prs_is_rpc = ?, prs_is_alive = ?, prs_is_ssl = ?, prs_is_main_net = ?, prs_is_validator = ?, prs_version = ?
			WHERE prs_id = ?`
	_, err = s.db.ExecContext(s.ctx, query, isRpc, isAlive, isSSL, isMainNet, isValidator, version, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) UpdatePeerNodePubkey(peerID int, nodePubkey string) (err error) {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `UPDATE peers SET prs_node_pubkey = ?
			WHERE prs_id = ?`
	_, err = s.db.ExecContext(s.ctx, query, nodePubkey, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) UpdatePeerIsOutdated(peerID int, isOutdated bool) (err error) {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `UPDATE peers SET prs_is_outdated = ?
			WHERE prs_id = ?`
	_, err = s.db.ExecContext(s.ctx, query, isOutdated, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetEndpoints(blockchainID, limit int, isRpc, isValidator *bool, asnCountries, versions, supportedMethods []string) (res []models.Endpoint, err error) {
	if blockchainID == 0 {
		return nil, fmt.Errorf("empty blockchainID")
	}
	for i := range asnCountries {
		asnCountries[i] = strings.ToUpper(asnCountries[i])
	}

	q := sq.Select(`ip_addr || ':' || prs_port AS endpoint,
		   prs_version,
		   prs_is_rpc,
		   prs_is_validator,
		   prs_is_ssl,
		   json_group_array(json_object('name', rpc_methods.mtd_name, 'response_time', rpc_peers_methods.pmd_response_time_ms)) AS supported_methods,
		   json_object('network', ntw_mask, 'isp', ntw_name, 'ntw_as', ntw_as, 'country',
									  json_object('alpha2', cnt_alpha2, 'alpha3', cnt_alpha3, 'name', cnt_name)) AS asn_info`).
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING (ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING (ntw_id)", geoNetworksTable)).
		LeftJoin(fmt.Sprintf("%s USING (cnt_id)", geoCountriesTable)).
		LeftJoin(fmt.Sprintf("%s USING (prs_id)", rpcPeersMethodsTable)).
		LeftJoin(fmt.Sprintf("%s USING (mtd_id)", rpcMethodsTable)).
		Where("prs_is_alive IS TRUE AND prs_is_main_net IS TRUE AND prs_is_outdated IS FALSE AND peers.blc_id = ?", blockchainID).
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
		q = q.Where(sq.Eq{"mtd_name": supportedMethods})
		q = q.Having(ArrayContain{"mtd_name": supportedMethods})
	}
	if len(versions) != 0 {
		// TODO: should be old logic for LikeAny: LIKE ANY (ARRAY [?])
		q = q.Where(sq.Eq{"prs_version": versions})
	}
	if limit != 0 {
		q = q.Limit(uint64(limit))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return res, err
	}

	log.Logger.Scanner.Debugf("quety %s", query)

	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint models.Endpoint
		var supportedMethodsStr, asnInfoStr string
		if err = rows.Scan(&endpoint.Endpoint, &endpoint.Version, &endpoint.IsRpc, &endpoint.IsValidator,
			&endpoint.IsSsl, &supportedMethodsStr, &asnInfoStr); err != nil {
			return res, err
		}

		err = json.Unmarshal([]byte(supportedMethodsStr), &endpoint.SupportedMethods)
		if err != nil {
			return res, fmt.Errorf("incorrect supportedMethods: %s, err: %s", supportedMethodsStr, err)
		}
		err = json.Unmarshal([]byte(asnInfoStr), &endpoint.AsnInfo)
		if err != nil {
			return res, fmt.Errorf("incorrect asnInfo: %s, err: %s", asnInfoStr, err)
		}

		res = append(res, endpoint)
	}

	return res, nil
}

func (s *Storage) GetPeers(isUniqIP bool, isAlive, isMainNet, isRpc *bool, blockchainID *int) (res []PeerWithIpAndBlockchain, err error) {
	if blockchainID != nil && *blockchainID == 0 {
		return nil, fmt.Errorf("empty blockchainID")
	}

	q := sq.Select(`prs_id, blc_id, blc_name, ip_id, ip_addr, prs_port, prs_version, prs_is_rpc, prs_is_alive, 
		prs_is_ssl, prs_is_main_net, prs_node_pubkey, prs_is_validator, prs_is_outdated`).
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING(ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING(blc_id)", blockchainsTable))
	if isAlive != nil {
		q = q.Where("prs_is_alive = ?", *isAlive)
	}
	if isMainNet != nil {
		q = q.Where("prs_is_main_net = ?", *isMainNet)
	}
	if isRpc != nil {
		q = q.Where("prs_is_rpc = ?", *isRpc)
	}
	if blockchainID != nil {
		q = q.Where("blc_id = ?", *blockchainID)
	}
	if isUniqIP {
		// act like distinct
		q = q.GroupBy("ip_id")
	}

	query, args, err := q.ToSql()
	if err != nil {
		return res, err
	}

	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var peer PeerWithIpAndBlockchain
		var addressStr string
		if err = rows.Scan(&peer.ID, &peer.BlockchainID, &peer.BlockchainName, &peer.IpID, &addressStr,
			&peer.Port, &peer.Version, &peer.IsRpc, &peer.IsAlive, &peer.IsSSL, &peer.IsMainNet, &peer.NodePubkey,
			&peer.IsValidator, &peer.IsOutdated); err != nil {
			return res, err
		}

		peer.Address = net.ParseIP(addressStr)
		res = append(res, peer)
	}

	return res, nil
}

func (s *Storage) GetExistentPeers(blockchainID int, ips []string) (res map[string]map[int]PeerWithIp, err error) {
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

	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	var peers []PeerWithIp
	res = make(map[string]map[int]PeerWithIp, len(peers)) // ip/port
	for rows.Next() {
		var peer PeerWithIp
		var addressStr string
		if err = rows.Scan(&peer.ID, &peer.Port, &peer.Version, &peer.NodePubkey, &addressStr); err != nil {
			return res, err
		}

		peer.Address = net.ParseIP(addressStr)
		if _, ok := res[peer.Address.String()]; !ok {
			res[peer.Address.String()] = make(map[int]PeerWithIp)
		}

		res[peer.Address.String()][peer.Port] = peer
	}

	return res, nil
}

func (s *Storage) GetStats() (res models.Stat, err error) {
	q := `SELECT COUNT(*)                                                   					AS total,
		   SUM(CASE WHEN prs_is_alive IS TRUE THEN 1 ELSE 0 END)     							AS alive,
		   SUM(CASE WHEN prs_is_rpc IS TRUE THEN 1 ELSE 0 END)       							AS rpc,
		   SUM(CASE WHEN prs_is_alive IS TRUE AND prs_is_validator IS true THEN 1 ELSE 0 END) 	AS validator
		FROM peers WHERE prs_is_main_net IS TRUE AND prs_is_outdated IS FALSE`

	err = s.db.QueryRowContext(s.ctx, q).Scan(&res.Total, &res.Alive, &res.Rpc, &res.Validator)
	if err != nil {
		return res, err
	}

	return res, nil
}
