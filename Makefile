.PHONY: build ingest run test

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= quotes
DB_CONN ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

build:
	go build -o bin/api ./cmd/api
	go build -o bin/ingest ./cmd/ingest

ingest:
	go run ./cmd/ingest $(ARGS)

run:
	go run ./cmd/api

test:
	go test ./...
