.PHONY: help build test lint migrate-up migrate-down

help:
	@echo "Available targets:"
	@echo "  make build        - Build neotex and neotexd binaries"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make migrate-up   - Run database migrations up"
	@echo "  make migrate-down - Run database migrations down"

build:
	@echo "Building neotex and neotexd..."
	go build -o bin/neotex ./cmd/neotex
	go build -o bin/neotexd ./cmd/neotexd

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linters..."
	go fmt ./...
	go vet ./...

migrate-up:
	@echo "Running migrations up..."
	@which migrate > /dev/null || (echo "golang-migrate not found. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	migrate -path migrations -database "postgres://neotex:neotex@localhost:5434/neotex?sslmode=disable" up

migrate-down:
	@echo "Running migrations down..."
	@which migrate > /dev/null || (echo "golang-migrate not found. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	echo "y" | migrate -path migrations -database "postgres://neotex:neotex@localhost:5434/neotex?sslmode=disable" down
