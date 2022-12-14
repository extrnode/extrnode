drop index public.peers_prs_is_alive_prs_is_main_net_blc_id_index;
create index peers_prs_is_alive_blc_id_index
    on public.peers (prs_is_alive, blc_id);

alter table public.peers
    drop column prs_is_main_net;
