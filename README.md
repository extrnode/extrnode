# Extrnode-go
The scanner is one of the Extra Node components.
It scans network and finds all kind of nodes, although the main focus is RPC.
The results get written to DB.

## Build and Deployment (local)
- install golang 1.19
- install and setup Clickhouse 23.1. Use migration for clickhouse [init-db.sh](build/clickhouse/init-db.sh)

### Build [scanner](cmd/scanner)
- install nmap from official site (https://nmap.org/)
- setup env vars [.env.scanner.example](.env.scanner.example)
- build
```
CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/scanner
```

### Build [proxy](cmd/proxy)
- add your certificates for https server in [creds](creds) dir (optional)
- setup env vars [.env.proxy.example](.env.proxy.example)
- build
```
CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/proxy
```

### Build [scanner api](cmd/scanner_api)
- add your certificates for https server in [creds](creds) dir (optional)
- setup env vars [.env.scanner_api.example](.env.scanner_api.example)
- build
```
CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/scanner_api
```

### Build [user service](cmd/user_api)
- add your certificates for https server in [creds](creds) dir (optional)
- add firebase.conf in [creds](creds) dir (required)
- setup env vars [.env.user_api.example](.env.user_api.example)
- install Postgresql 11
- create a Postgresql DB
- build
```
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/user_api
```

### Run
- `./scanner` to start collecting new nodes
- `./scanner_api` to start scanner api server
- `./user_api` to start user service
- `./proxy` to start proxy balancer

## Build and Deployment (via [docker-compose.yml](docker-compose.yml))
- add your certificates for https server in [creds](creds) dir (optional)
- add firebase.conf in [creds](creds) dir (required)
- place filled [.env](.env.example) file into project root folder
- build:
```
make build
```
- run:
```
make dev
make start
```
- to stop containers run:
```
make stop
```

## Programs command line options
```
-log string
        log level [debug|info|warn|error|crit] (default "debug")
```

## API documentation
Api documentation for swagger located at [swagger.json](swagger/swagger.json)

## DB migrations
All migrations are embedded and tracked by program itself. You have not to track the migrations. All relations, schemes, indexes, so on will be
created within first time run of the data loader

## Running Tests
### Generate mocks
    go install github.com/golang/mock/mockgen@v1.6.0
    make mocks
### Run tests
    make test
### Run for development
Will run dependencies like clickhouse and postgres with ports open locally

    make dev