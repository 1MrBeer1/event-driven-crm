APP_SERVICES := gateway lead-service customer-service notification-service

.PHONY: up down build run migrate test test-integration lint tidy

up:
	docker compose up -d --build

down:
	docker compose down

build:
	go build ./cmd/gateway ./cmd/lead-service ./cmd/customer-service ./cmd/notification-service

run:
	docker compose up --build

migrate:
	docker compose run --rm migrate

test:
	go test ./...

test-integration:
	go test -tags=integration ./tests/integration

lint:
	gofmt -w .
	go vet ./...

tidy:
	go mod tidy

