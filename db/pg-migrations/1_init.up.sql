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

create unique index if not exists methods__uniq_index
	on rpc.methods (blc_id, mtd_name);

create table if not exists geo.countries
(
	cnt_id serial not null
		constraint countries_pk
			primary key,
	cnt_alpha2 char(2) not null,
	cnt_alpha3 char(3) not null,
	cnt_name varchar(64) not null
);

create table if not exists geo.networks
(
	ntw_id serial not null
		constraint networks_pk
			primary key,
	cnt_id integer
		constraint networks_countries_cnt_id_fk
			references geo.countries
				on update cascade on delete restrict,
	ntw_mask inet not null,
	ntw_as integer,
	ntw_name varchar(128) not null
);

create unique index if not exists networks_ntw_mask_uindex
	on geo.networks (ntw_mask);

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

create unique index if not exists ips_ip_id_uindex
	on ips (ip_id);

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
	prs_port smallint not null,
	prs_version varchar(8),
	prs_is_rpc boolean,
	prs_is_alive boolean
);

comment on table peers is 'todo:
source of data (rpc fetch, logs) ';

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
	constraint peers_methods_pk
		primary key (prs_id, mtd_id)
);

create table if not exists scanner.peers
(
	spr serial not null
		constraint peers_pk
			primary key,
	prs_id integer not null
		constraint peers_peers_prs_id_fk
			references peers
				on update cascade on delete restrict,
	spr_date timestamp not null,
	spr_time_connect integer,
	spr_is_alive boolean default false not null
);

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
	smt_time_connect integer,
	smt_time_response integer,
	smt_response_code smallint,
	smt_reponse_valid boolean
);

create unique index if not exists countries_cnt_alpha2_uindex
	on geo.countries (cnt_alpha2);

create unique index if not exists countries_cnt_alpha3_uindex
	on geo.countries (cnt_alpha3);

