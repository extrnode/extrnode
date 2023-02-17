#!/bin/bash
set -e

clickhouse client -n <<-EOSQL
    CREATE DATABASE IF NOT EXISTS extrnode;

    CREATE TABLE IF NOT EXISTS extrnode.stats(
        user_uuid String,
        request_id UUID,
        status UInt16,
        execution_time_ms Int64,
        endpoint String,
        attempts UInt8,
        response_time_ms Int64,
        rpc_error_code String,
        user_agent String,
        rpc_method String,
        rpc_request_data String,
    ) ENGINE = ReplacingMergeTree()
    ORDER BY (user_uuid, request_id);

    CREATE TABLE IF NOT EXISTS extrnode.scanner_methods(
        server_id String,
        time DateTime,
        peer String,
        method String,
        time_connect_ms Int64,
        time_response_ms Int64,
        response_code UInt16,
        response_valid Bool
    ) ENGINE = ReplacingMergeTree()
    ORDER BY (server_id, time, peer, method);

    CREATE TABLE IF NOT EXISTS extrnode.scanner_peers(
        server_id String,
        time DateTime,
        peer String,
        time_connect_ms Int64,
        is_alive Bool
    ) ENGINE = ReplacingMergeTree()
    ORDER BY (server_id, time, peer);

EOSQL