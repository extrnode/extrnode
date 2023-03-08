.PHONY: build start dev stop test mocks lint clickhouse

SHELL := /bin/bash

export APP_NAME=extrnode-go
export APP_VERSION=latest

build:
	@echo "building ${APP_NAME} with version ${APP_VERSION}"
	@echo "building docker image ${APP_IMAGE}"
	@docker build -f Dockerfile . -t ${APP_NAME}:${APP_VERSION}

start:
	@docker-compose up -d

dev: clickhouse
	@docker-compose -f docker-compose-postgres.yml up -d postgres

clickhouse:
	@docker-compose -f docker-compose-clickhouse.yml up -d clickhouse

stop:
	@docker-compose stop

test:
	@go test -v ./...

mocks:
	# go install github.com/golang/mock/mockgen@v1.6.0
	@go generate ./...
lint:
	# go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	@golangci-lint run