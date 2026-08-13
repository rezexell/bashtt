APP_NAME := bashtt
AGENT_NAME := agent

DATABASE_URL ?= postgres://bashtt:bashtt@localhost:5432/bashtt?sslmode=disable

MIGRATIONS_DIR := migrations

.PHONY: help
.PHONY: up
.PHONY: down
.PHONY: restart
.PHONY: logs
.PHONY: migrate-up
.PHONY: migrate-down
.PHONY: migrate-version
.PHONY: migrate-force
.PHONY: build
.PHONY: build-agent
.PHONY: build-server
.PHONY: run
.PHONY: test
.PHONY: tidy
.PHONY: ssh-up
.PHONY: ssh-down

help:
	@echo "Available commands:"
	@echo ""
	@echo "  make up              Start PostgreSQL"
	@echo "  make down            Stop PostgreSQL"
	@echo "  make restart         Restart PostgreSQL"
	@echo "  make logs            Show PostgreSQL logs"
	@echo ""
	@echo "  make migrate-up      Apply migrations"
	@echo "  make migrate-down    Roll back last migration"
	@echo "  make migrate-version Show migration version"
	@echo ""
	@echo "  make build           Build server and agent"
	@echo "  make build-server    Build server"
	@echo "  make build-agent     Build agent"
	@echo "  make run             Run server"
	@echo ""
	@echo "  make test            Run Go tests"
	@echo "  make tidy            Run go mod tidy"
	@echo ""
	@echo "  make ssh-up          Start SSH test container"
	@echo "  make ssh-down        Stop SSH test container"

up:
	docker compose up -d postgres

down:
	docker compose down

restart:
	docker compose restart postgres

logs:
	docker compose logs -f postgres

migrate-up:
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		up

migrate-down:
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		down 1

migrate-version:
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		version

migrate-force:
	@read -p "Migration version: " version; \
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		force $$version

build: build-server build-agent

build-server:
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build \
		-o bin/server \
		./cmd/server

build-agent:
	mkdir -p bin
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build \
		-o bin/agent \
		./cmd/agent

run:
	go run ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

ssh-up:
	docker compose -f docker-compose.test.yml up -d --build

ssh-down:
	docker compose -f docker-compose.test.yml down
