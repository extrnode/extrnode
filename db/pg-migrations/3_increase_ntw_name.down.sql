alter table geo.networks
    alter column ntw_name type varchar(128) using ntw_name::varchar(128);

