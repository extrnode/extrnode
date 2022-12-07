# Extrnode-go

### Serve service
Place filled [config.yml](config_example.yml), [.env](.env.example) file into `.secrets` folder and Run

    docker-compose up -d extrnode-scanner
### Serve API documentation
Serve API documentation at `:8082` port

    docker-compose up -d swagger-ui
### Running Tests
#### Generate mocks
    go install github.com/golang/mock/mockgen@v1.6.0
    make mocks
### Run tests
    make test
### Run for development
Will run dependencies like postgres with ports open locally

    make dev