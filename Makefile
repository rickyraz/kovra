.PHONY: setup setup-week2 migrate migrate-down migrate-status run test test-coverage lint generate clean

# Default database URL
DATABASE_URL ?= postgresql://kovra:kovra_dev@localhost:5432/kovra?sslmode=disable

# Development setup (Week 1: single TigerBeetle node)
setup:
	docker-compose up -d postgres redis tigerbeetle
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "Infrastructure ready!"

# Week 2 setup (3-node TigerBeetle cluster)
setup-week2:
	docker-compose --profile week2 up -d postgres redis tigerbeetle-0 tigerbeetle-1 tigerbeetle-2
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Week 2 infrastructure ready!"

# Stop all services
down:
	docker-compose --profile week1 --profile week2 down

# Database migrations
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-down-all:
	goose -dir migrations postgres "$(DATABASE_URL)" down-to 0

migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create:
	@read -p "Migration name: " name; \
	goose -dir migrations create $$name sql

# Run application
run:
	go run cmd/api/main.go

# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	go test -v -tags=integration ./...

# Week demos
demo-week1:
	go test -v -run TestWeek1Demo ./e2e/...

demo-week2:
	go test -v -run TestWeek2Demo ./e2e/...

# Code generation
generate:
	sqlc generate

# Linting
lint:
	golangci-lint run

# Build
build:
	go build -o bin/kovra-api cmd/api/main.go

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install tools
tools:
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Tidy dependencies
tidy:
	go mod tidy
