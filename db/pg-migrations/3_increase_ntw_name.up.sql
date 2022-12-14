alter table geo.networks
    alter column ntw_name type varchar(256) using ntw_name::varchar(256);
