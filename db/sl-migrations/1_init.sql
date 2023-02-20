-- +migrate Up
create table if not exists blockchains
(
    blc_id integer not null on conflict rollback
        constraint blockchains_pk
            primary key autoincrement,
    blc_name varchar(32) not null on conflict rollback
        constraint blockchains_blc_name_uindex
            unique
);

create table if not exists rpc_methods
(
    mtd_id         integer                not null on conflict rollback
        constraint rpc_methods_pk
            primary key autoincrement,
    blc_id         integer                not null on conflict rollback
        constraint rpc_methods_blockchains_blc_id_fk
            references blockchains
            on update cascade on delete restrict,
    mtd_name       varchar(64)            not null on conflict rollback,
    mtd_created_at NUMERIC default (DATETIME('now')) not null on conflict rollback
);
create index rpc_methods_blc_id_index
    on rpc_methods (blc_id);
create unique index if not exists rpc_methods_blc_id_mtd_name_uniq_index
    on rpc_methods (blc_id, mtd_name);

create table if not exists geo_countries
(
    cnt_id integer                      not null on conflict rollback
        constraint geo_countries_pk
            primary key autoincrement,
    cnt_alpha2 char(2) not null on conflict rollback,
    cnt_alpha3 char(3) not null on conflict rollback,
    cnt_name varchar(64) not null on conflict rollback
);
create unique index if not exists geo_countries_cnt_alpha2_uindex
    on geo_countries (cnt_alpha2);
create unique index if not exists geo_countries_cnt_alpha3_uindex
    on geo_countries (cnt_alpha3);

create table if not exists geo_networks
(
    ntw_id   integer      not null on conflict rollback
        constraint geo_networks_pk
            primary key autoincrement,
    cnt_id   integer      not null on conflict rollback
        constraint geo_networks_geo_countries_cnt_id_fk
            references geo_countries
            on update cascade on delete restrict,
    ntw_mask varchar(43)  not null on conflict rollback,
    ntw_as   integer,
    ntw_name varchar(256) not null on conflict rollback
);
create index geo_networks_cnt_id_index
    on geo_networks (cnt_id);
create unique index if not exists geo_networks_cnt_id_ntw_mask_uindex
    on geo_networks (cnt_id, ntw_mask);

create table if not exists ips
(
    ip_id integer not null on conflict rollback
        constraint ips_pk
            primary key autoincrement,
    ntw_id integer not null on conflict rollback
        constraint ips_geo_networks_ntw_id_fk
            references geo_networks
            on update cascade on delete restrict,
    ip_addr varchar(39) not null on conflict rollback
);
create unique index if not exists ips_ip_addr_uindex
    on ips (ip_addr);
create index ips_ntw_id_index
    on ips (ntw_id);

create table if not exists peers
(
    prs_id integer not null on conflict rollback
        constraint peers_pk
            primary key autoincrement,
    blc_id integer not null on conflict rollback
        constraint peers_blockchains_blc_id_fk
            references blockchains
            on update cascade on delete restrict,
    ip_id integer not null on conflict rollback
        constraint peers_ips_ip_id_fk
            references ips
            on update cascade on delete restrict,
    prs_port integer not null on conflict rollback,
    prs_version varchar(8) not null on conflict rollback,
    prs_is_rpc boolean default false not null on conflict rollback,
    prs_is_alive boolean default false not null on conflict rollback,
    prs_is_ssl boolean default false not null on conflict rollback,
    prs_is_main_net boolean default true not null on conflict rollback,
    prs_node_pubkey varchar(44) default '' not null on conflict rollback,
    prs_is_validator boolean default false not null on conflict rollback,
    prs_is_outdated boolean default false not null on conflict rollback
);
create unique index if not exists peers_blc_id_ip_id_prs_port_uindex
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

create table if not exists rpc_peers_methods
(
    prs_id integer not null on conflict rollback
        constraint peers_methods_peers_prs_id_fk
            references peers
            on update cascade on delete restrict,
    mtd_id integer not null on conflict rollback
        constraint peers_methods_methods_mtd_id_fk
            references rpc_methods
            on update cascade on delete restrict,
    pmd_response_time_ms integer default 0 not null,
    constraint rpc_peers_methods_pk
        primary key (prs_id, mtd_id)
);

-- default data
INSERT INTO blockchains (blc_name) VALUES ('solana');

INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getAccountInfo');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'sendTransaction');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getSignaturesForAddress');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getLatestBlockhash');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getSlot');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getTransaction');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getInflationReward');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getProgramAccounts');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getSignatureStatuses');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getTokenAccountBalance');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getMultipleAccounts');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getEpochInfo');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getBalance');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getRecentPerformanceSamples');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getVoteAccounts');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getInflationRate');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getSupply');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getBlockTime');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getBlockHeight');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getMinimumBalanceForRentExemption');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'isBlockhashValid');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getTransactionCount');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid  FROM blockchains WHERE blc_name = 'solana'), 'getTokenAccountsByOwner');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getBlock');
INSERT INTO rpc_methods (blc_id, mtd_name) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), 'getVersion');

-- initial solana host
INSERT INTO geo_countries (cnt_alpha2, cnt_alpha3, cnt_name) VALUES ('US', 'USA', 'United States');
INSERT INTO geo_networks (cnt_id, ntw_mask, ntw_as, ntw_name) VALUES ((SELECT rowid FROM geo_countries WHERE cnt_alpha2 = 'US'), '107.155.92.0/24', 29802, 'HVC-AS, US');
INSERT INTO ips (ntw_id, ip_addr) VALUES ((SELECT rowid FROM geo_networks WHERE ntw_mask = '107.155.92.0/24'), '107.155.92.114');
INSERT INTO peers (blc_id, ip_id, prs_port, prs_version, prs_is_rpc, prs_is_alive, prs_is_ssl, prs_is_main_net, prs_node_pubkey, prs_is_validator) VALUES ((SELECT rowid FROM blockchains WHERE blc_name = 'solana'), (SELECT rowid FROM ips WHERE ntw_id = (SELECT rowid FROM geo_networks WHERE ntw_mask = '107.155.92.0/24') AND ip_addr = '107.155.92.114'), 80, '1.13.5', true, true, false, true, '', false);


-- +migrate Down
drop table if exists blockchains;
drop table if exists geo_countries;
drop table if exists geo_networks;
drop table if exists ips;
drop table if exists peers;
drop table if exists rpc_methods;
drop table if exists rpc_peers_methods;
