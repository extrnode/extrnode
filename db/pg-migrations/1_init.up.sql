create table if not exists blockchains
(
	blc_id smallserial not null
		constraint blockchains_pk
			primary key,
	blc_name varchar default 32 not null
);
create unique index if not exists blockchains_blc_name_uindex
	on blockchains (blc_name);

create table if not exists rpc.methods
(
	mtd_id serial not null
		constraint methods_pk
			primary key,
	blc_id integer not null
		constraint methods_blockchains_blc_id_fk
			references blockchains
				on update cascade on delete restrict,
	mtd_name varchar(32) not null,
	mtd_created_at timestamp default now() not null
);
create unique index if not exists methods_blc_id_mtd_name_uniq_index
	on rpc.methods (blc_id, mtd_name);
create index methods_blc_id_index
    on rpc.methods (blc_id);

create table if not exists geo.countries
(
	cnt_id serial not null
		constraint countries_pk
			primary key,
	cnt_alpha2 char(2) not null,
	cnt_alpha3 char(3) not null,
	cnt_name varchar(64) not null
);
create unique index if not exists countries_cnt_alpha2_uindex
    on geo.countries (cnt_alpha2);
create unique index if not exists countries_cnt_alpha3_uindex
    on geo.countries (cnt_alpha3);

create table if not exists geo.networks
(
	ntw_id serial not null
		constraint networks_pk
			primary key,
	cnt_id integer
		constraint networks_countries_cnt_id_fk
			references geo.countries
				on update cascade on delete restrict,
	ntw_mask cidr not null,
	ntw_as integer,
	ntw_name varchar(256) not null
);
create unique index if not exists networks_cnt_id_ntw_mask_uindex
    on geo.networks (cnt_id, ntw_mask);
create index networks_cnt_id_index
    on geo.networks (cnt_id);

create table if not exists ips
(
	ip_id serial not null
		constraint ips_pk
			primary key,
	ntw_id integer
		constraint ips_networks_ntw_id_fk
			references geo.networks
				on update cascade on delete restrict,
	ip_addr inet not null
);
comment on table ips is 'Todo:
- attemps to detect network
- next attempt ';
create unique index if not exists ips_ip_addr_uindex
	on ips (ip_addr);
create index ips_ntw_id_index
    on ips (ntw_id);

create table if not exists peers
(
	prs_id serial not null
		constraint peers_pk
			primary key,
	blc_id integer not null
		constraint peers_blockchains_blc_id_fk
			references blockchains
				on update cascade on delete restrict,
	ip_id integer not null
		constraint peers_ips_ip_id_fk
			references ips
				on update cascade on delete restrict,
	prs_port integer not null,
	prs_version varchar(8) not null,
	prs_is_rpc boolean default false not null,
	prs_is_alive boolean default false not null,
	prs_is_ssl boolean default false not null,
    prs_is_main_net boolean default true not null,
    prs_node_pubkey varchar(44) default '' not null,
    prs_is_validator boolean default false not null
);
comment on table peers is 'todo:
source of data (rpc fetch, logs) ';
create unique index peers_blc_id_ip_id_prs_port_uindex
    on peers (blc_id, ip_id, prs_port);
create index peers_blc_id_index
    on peers (blc_id);
create index peers_ip_id_index
    on peers (ip_id);
create index peers_prs_version_index
    on peers (prs_version);
create index peers_prs_is_alive_prs_is_main_net_blc_id_index
    on peers (prs_is_alive, prs_is_main_net, blc_id);
create index peers_prs_is_rpc_index
    on peers (prs_is_rpc);

create table if not exists rpc.peers_methods
(
	prs_id integer not null
		constraint peers_methods_peers_prs_id_fk
			references peers
				on update cascade on delete restrict,
	mtd_id integer not null
		constraint peers_methods_methods_mtd_id_fk
			references rpc.methods
				on update cascade on delete restrict,
    pmd_response_time_ms integer default 0 not null,
	constraint peers_methods_pk
		primary key (prs_id, mtd_id)
);

create table if not exists scanner.peers
(
	spr_id serial not null
		constraint peers_pk
			primary key,
	prs_id integer not null
		constraint peers_peers_prs_id_fk
			references peers
				on update cascade on delete restrict,
	spr_date timestamp not null,
	spr_time_connect_ms integer,
	spr_is_alive boolean default false not null
);
create unique index peers_prs_id_spr_date_uindex
    on scanner.peers (prs_id, spr_date);
create index peers_prs_id_index
    on scanner.peers (prs_id);

create table if not exists scanner.methods
(
	smt_id serial not null
		constraint methods_pk
			primary key,
	prs_id integer not null
		constraint methods_peers_prs_id_fk
			references peers
				on update cascade on delete restrict,
	mtd_id integer not null
		constraint methods_methods_mtd_id_fk
			references rpc.methods
				on update cascade on delete restrict,
	smt_date timestamp not null,
	smt_time_connect_ms integer,
	smt_time_response_ms integer,
	smt_response_code smallint,
	smt_response_valid boolean
);
create unique index methods_prs_id_mtd_id_smt_date_uindex
    on scanner.methods (prs_id, mtd_id, smt_date);
create index methods_mtd_id_index
    on scanner.methods (mtd_id);
create index methods_prs_id_index
    on scanner.methods (prs_id);
create index methods_smt_date_index
    on scanner.methods (smt_date desc);


-- default data
INSERT INTO blockchains (blc_name) VALUES ('solana');

INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getAccountInfo');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'sendTransaction');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getSignaturesForAddress');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getLatestBlockhash');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getSlot');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getTransaction');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getInflationReward');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getProgramAccounts');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getSignatureStatuses');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getTokenAccountBalance');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getMultipleAccounts');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getEpochInfo');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getBalance');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getRecentPerformanceSamples');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getVoteAccounts');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getInflationRate');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getSupply');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getBlockTime');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getBlockHeight');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getMinimumBalanceForRentExemptio');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'isBlockhashValid');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getTransactionCount');
INSERT INTO rpc.methods (blc_id, mtd_name) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), 'getTokenAccountsByOwner');

-- initial solana host http://api.mainnet-beta.solana.com
INSERT INTO geo.countries (cnt_alpha2, cnt_alpha3, cnt_name) VALUES ('NL', 'NLD', 'Netherlands');
INSERT INTO geo.networks (cnt_id, ntw_mask, ntw_as, ntw_name) VALUES ((SELECT cnt_id FROM geo.countries WHERE cnt_alpha2 = 'NL'), '178.237.58.0/24', 56504, 'HOSTCIRCLE-L-, NL');
INSERT INTO ips (ntw_id, ip_addr) VALUES ((SELECT ntw_id FROM geo.networks WHERE ntw_mask = '178.237.58.0/24'), '178.237.58.144');
INSERT INTO peers (blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, prs_is_main_net) VALUES ((SELECT blc_id FROM blockchains WHERE blc_name = 'solana'), (SELECT ip_id FROM ips WHERE ntw_id = (SELECT ntw_id FROM geo.networks WHERE ntw_mask = '178.237.58.0/24') AND ip_addr = '178.237.58.144'), 80, '1.14.10', true, true, false, true);
