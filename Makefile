.PHONY: run-api run-scheduler run-worker test lint vet fmt migrate-up migrate-down docker-up docker-down web-dev web-build web-test

run-api:
	go run ./apps/api/cmd/api

run-scheduler:
	go run ./apps/scheduler/cmd/scheduler

run-worker:
	go run ./apps/worker/cmd/worker

run-monitor-engine:
	go run ./apps/monitor-engine/cmd/monitor-engine

run-alert-engine:
	go run ./apps/alert-engine/cmd/alert-engine

run-probe-gateway:
	go run ./apps/probe-gateway/cmd/probe-gateway

run-agent-gateway:
	go run ./apps/agent-gateway/cmd/agent-gateway

run-metric-processor:
	go run ./apps/metric-processor/cmd/metric-processor

test:
	go test ./... -race -cover

vet:
	go vet ./...

fmt:
	gofmt -w ./apps ./packages

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
