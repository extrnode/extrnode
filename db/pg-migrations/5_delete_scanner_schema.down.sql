CREATE SCHEMA IF NOT EXISTS scanner;

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