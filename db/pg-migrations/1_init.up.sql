create table public.users
(
    usr_id        bigserial
        constraint users_pk
            primary key,
    usr_provider_id      varchar(128) not null,
    usr_api_token        uuid         not null
);
create unique index users_usr_provider_id_uindex
    on public.users (usr_provider_id);
create unique index users_usr_api_token_uindex
    on public.users (usr_api_token);