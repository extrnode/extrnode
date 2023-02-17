# Extrnode-go
The scanner is one of the Extra Node components.
It scans network and finds all kind of nodes, although the main focus is RPC.
The results get written to DB.

## Build and Deployment (local)
- install golang 1.19
- install Postgresql 11
- create a Postgresql DB
- install Clickhouse 23.1
- use migration for clickhouse [init-db.sh](build/clickhouse/init-db.sh)
- install nmap from official site (https://nmap.org/)
- setup env vars from [.env.example](.env.example) file
- compile programs:

```
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/scanner
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/api
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/proxy
```
- run ./scanner to start collecting new nodes
- run ./api to start api server
- run ./proxy to start proxy balancer

## Build and Deployment (via [docker-compose.yml](docker-compose.yml))
- place filled [.env](.env.example) file into project root folder
- add your certificates for https server in [creds](creds) dir (optional)
- add firebase.conf in [creds](creds) dir (required)
- build:
```
make build
```
- run:
```
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
Will run dependencies like postgres with ports open locally

    make dev