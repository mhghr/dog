.PHONY: run-api run-scheduler run-worker test lint vet fmt migrate-up migrate-down docker-up docker-down web-dev web-build web-test

run-api:
	go run ./cmd/api

run-scheduler:
	go run ./cmd/scheduler

run-worker:
	go run ./cmd/worker

test:
	go test ./... -race -cover

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

lint:
	golangci-lint run

migrate-up:
	migrate \
		-path migrations \
		-database "$(DATABASE_URL)" \
		up

migrate-down:
	migrate \
		-path migrations \
		-database "$(DATABASE_URL)" \
		down 1

docker-up:
	docker compose up --build

docker-down:
	docker compose down

web-dev:
	pnpm --filter web dev

web-build:
	pnpm --filter web build

web-test:
	pnpm --filter web test
