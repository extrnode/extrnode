alter table peers
    add prs_node_pubkey varchar(44) default '' not null;

alter table peers
    add prs_is_validator bool default false not null;
