package postgres

import (
	"fmt"
	"net"

	sq "github.com/Masterminds/squirrel"
)

type (
	Peer struct {
		Port          int       `pg:"prs_port"`
		Version       string    `pg:"prs_version"`
		IsRpc         bool      `pg:"prs_is_rpc"`
		IsAlive       bool      `pg:"prs_is_alive"`
		IsSSL         bool      `pg:"prs_is_ssl"`
		IsMainNet     bool      `pg:"prs_is_main_net"`
		NodePubkey    string    `pg:"prs_node_pubkey"`
		IsValidator   bool      `pg:"prs_is_validator"`
		Address       net.IP    `pg:"ip_addr"`
		NetworkMask   net.IPNet `pg:"ntw_mask"`
		NetworkAs     int       `pg:"ntw_as"`
		NetworkName   string    `pg:"ntw_name"`
		CountryAlpha2 string    `pg:"cnt_alpha2"`
		CountryAlpha3 string    `pg:"cnt_alpha3"`
		CountryName   string    `pg:"cnt_name"`
	}
)

const (
	peersTable        = "peers"
	ipsTable          = "ips"
	geoNetworksTable  = "geo.networks"
	geoCountriesTable = "geo.countries"
)

func (p *Storage) GetEndpoints() (res []Peer, err error) {
	q := sq.Select(`prs_port,
			prs_version,
			prs_is_rpc,
			prs_is_alive,
			prs_is_ssl,
			prs_is_main_net,
			prs_node_pubkey,
			prs_is_validator,
			ip_addr,
		   ntw_mask,
		   ntw_as,
		   ntw_name,
		   cnt_alpha2,
		   cnt_alpha3,
		   cnt_name`).
		From(peersTable).
		LeftJoin(fmt.Sprintf("%s USING (ip_id)", ipsTable)).
		LeftJoin(fmt.Sprintf("%s USING (ntw_id)", geoNetworksTable)).
		LeftJoin(fmt.Sprintf("%s USING (cnt_id)", geoCountriesTable)).
		Where("prs_is_main_net IS TRUE").
		OrderByClause("prs_id")

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
