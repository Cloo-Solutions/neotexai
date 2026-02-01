.PHONY: help build test lint migrate-up migrate-down sync-skill

help:
	@echo "Available targets:"
	@echo "  make build        - Build neotex and neotexd binaries"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make migrate-up   - Run database migrations up"
	@echo "  make migrate-down - Run database migrations down"
	@echo "  make sync-skill   - Sync SKILL.md from packages/skill to CLI embed"

sync-skill:
	@echo "Syncing skills..."
	cp packages/skill/SKILL.md internal/cli/client/skill_embed.md
	cp packages/skill/neotex-init.md internal/cli/client/skill_init_embed.md

build: sync-skill
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
