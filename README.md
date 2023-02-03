# Extrnode-go
## Why are RPCs so important?
A cryptocurrency wallet does not actually connect to the blockchain. It simply turns actions in the interface into code and then sends it to one of the nodes to be executed and included in the blockchain.

Wallets and other applications send code to pre-selected RPC nodes. If they stop responding and accepting requests, the applications won't work.

Hosting a Solana node is expensive, starting at $1,000 per month, so dApp developers often send requests to public RPC nodes. Those public RPCs, however, are usually hosted by centralized providers like Google Cloud, Amazon Web Services, and Hetzner.

A case in point: in early November, Hetzner discontinued all Solana nodes on its servers, which comprised a whopping 22% of the overall number of nodes. The network survived, but many apps crashed as their selected RPC nodes went offline.

![extrnode1.png](public/extrnode1.png)

This story shows that trusting only one RPC on a centrally hosted service is dangerous. A dApp developer can reduce the chances of failure with a script, module, or standalone app that automatically switches to a spare RPC endpoint in case of any problem with the primary node. But what if the alternate one fails too? This is the problem we are going to solve with extrnode.

## Why are RPCs so important?
The RPC layer is centralized. Many dApps are connected to RPC nodes run by a handful of major providers and hosted on the same servers. If something happens to the hosting providers or the nodes, the dApps will lose their connection to the blockchain and stop working.

Solana's developers needed a tool to switch their application to a spare RPC node automatically in case of problems with the node in use. So we developed extrnode’s public load balancer, a solution that automatically reroutes application requests to one of the working RPC nodes from a vast cluster.

## How can extrnode help dApp developers?
Developers can be confident that their applications will always have access to an RPC, and users can use those apps without errors or delays. To achieve this without extrnode, developers would have to ask users to manually switch to other RPCs. Building a custom load balancer can only be done by a large team: it takes money, expertise, and active assistance from validators and infrastructure providers.

Using extrnode developers will need to send requests to extrnode's RPC endpoint for the load balancer to reroute them to other RPCs.

![extrnode2.png](public/extrnode2.png)

## Build and Deployment (local)
- install golang 1.19
- install Postgresql 11
- create a Postgresql DB 
- install nmap from official site (https://nmap.org/)
- setup env vars from [.env.example](.env.example) file
- compile programs:

```
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/scanner
CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/api
```
- run ./scanner to start collecting new nodes
- run ./api to start api server

## Build and Deployment (via [docker-compose.yml](docker-compose.yml))
- place filled [.env](.env.example) file into project root folder
- add your certificates for https server in `certs` dir (optional)
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

## Enviroment variables description
```
# scanner
# how many threads the scanner uses.
SCANNER_THREADS_NUM=20

# api 
# servert port
API_PORT=8000
# port for prometheus metrics (optional; 0 or empty value - disable metrics)
API_METRICS_PORT=9099
# path to certs for https (optional)
API_CERT_FILE=certs/api.pem
# failover endpoints for proxy. Json encoded object array (optional)
API_FAILOVER_ENDPOINTS=[{"url":"http://127.0.0.1:8001","reqLimitHourly":1},{"url":"http://127.0.0.1","reqLimitHourly":2}]

# database connection properties
PG_HOST=localhost
PG_PORT=5432
PG_USER=extrnode
PG_PASS=somepass
# database name
PG_DB=extrnode
# path to migrations dir
PG_MIGRATIONS_PATH=db/pg-migrations
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