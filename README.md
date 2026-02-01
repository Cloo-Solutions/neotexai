# NeotexAI

Agent-first knowledge management for AI agents.

[![License](https://img.shields.io/badge/license-OSAASy-blue)](LICENSE)
[![Build](https://github.com/cloo-solutions/neotexai/actions/workflows/release.yml/badge.svg)](https://github.com/cloo-solutions/neotexai/actions/workflows/release.yml)
[![Docker](https://ghcr-badge.egpl.dev/cloo-solutions/neotexai/latest_tag?trim=major&label=docker)](https://ghcr.io/cloo-solutions/neotexai)

## What is NeotexAI?

NeotexAI is a knowledge management system designed for AI agents:
- **Semantic search** over organizational knowledge
- **Immutable versioning** for all content
- **Asset management** with S3-compatible storage
- **Multi-tenant** organization support

## Install (Client CLI)

**One-liner:**
```bash
curl -fsSL https://raw.githubusercontent.com/cloo-solutions/neotexai/main/scripts/install.sh | sh
```

**With Go:**
```bash
go install github.com/cloo-solutions/neotexai/cmd/neotex@latest
```

**Download binary:**
See [Releases](https://github.com/cloo-solutions/neotexai/releases)

## Claude Code Skills

```bash
npx skills add https://github.com/cloo-solutions/neotexai/tree/main/skills/neotex
npx skills add https://github.com/cloo-solutions/neotexai/tree/main/skills/neotex-init
```

## Server Setup (Docker)

```bash
docker run -d \
  -p 8080:8080 \
  -e NEOTEX_DATABASE_URL=postgres://user:pass@host:5432/neotex \
  -e NEOTEX_OPENAI_API_KEY=sk-... \
  ghcr.io/cloo-solutions/neotexai:latest
```

See [Configuration](#configuration) for all environment variables.

## Usage

```bash
# Initialize project
neotex init --api-key <your-key>

# Pull knowledge manifest
neotex pull

# Search knowledge
neotex search "how to deploy"

# Get specific item
neotex get <id>
```

## Configuration

### Server (`neotexd`)

| Variable | Required | Description |
|----------|----------|-------------|
| `NEOTEX_DATABASE_URL` | Yes | PostgreSQL connection string |
| `NEOTEX_OPENAI_API_KEY` | Yes | OpenAI API key for embeddings |
| `NEOTEX_S3_ENDPOINT` | No | S3-compatible storage endpoint |
| `NEOTEX_S3_BUCKET` | No | Bucket name for assets |
| `SENTRY_DSN` | No | Sentry DSN for error tracking |

### Client (`neotex`)

| Variable | Required | Description |
|----------|----------|-------------|
| `NEOTEX_API_KEY` | Yes | API key from server |
| `NEOTEX_API_URL` | No | Server URL (default: http://localhost:8080) |

## Development

```bash
# Clone and setup
git clone https://github.com/cloo-solutions/neotexai.git
cd neotexai
./scripts/setup.sh

# Run tests
make test

# Build binaries
make build
```

## License

OSAASy License - see [LICENSE](LICENSE)
