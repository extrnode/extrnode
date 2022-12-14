alter table public.peers
    add prs_is_main_net boolean default true not null;

drop index public.peers_prs_is_alive_blc_id_index;
create index peers_prs_is_alive_prs_is_main_net_blc_id_index
    on public.peers (prs_is_alive, prs_is_main_net, blc_id);
