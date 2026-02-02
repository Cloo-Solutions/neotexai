# NeotexAI

Share organizational knowledge across your team's AI agents.

[![License](https://img.shields.io/badge/license-OSAASy-blue)](LICENSE)
[![Build](https://github.com/cloo-solutions/neotexai/actions/workflows/release.yml/badge.svg)](https://github.com/cloo-solutions/neotexai/actions/workflows/release.yml)
[![Docker](https://ghcr-badge.egpl.dev/cloo-solutions/neotexai/latest_tag?trim=major&label=docker)](https://ghcr.io/cloo-solutions/neotexai)

## What is NeotexAI?

NeotexAI lets teams collaborate on organizational knowledge that AI agents can access. When one team member documents a convention, decision, or pattern, everyone's AI agents can find and use it.

- **Team knowledge sharing** - Guidelines, decisions, and patterns sync across your organization
- **AI-native search** - Agents find relevant context through semantic search
- **Assets included** - Share reference images, mockups, and files alongside documentation
- **Version history** - Track how knowledge evolves over time

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

# Search knowledge and assets (hybrid by default)
neotex search "how to deploy"

# Search with filters (inline or flags)
neotex search "type:guideline status:active path:backend how to deploy"
neotex search "login mockup" --source asset --mode lexical --limit 10
neotex search "postgres migration" --exact

# Get specific item (optionally link to search for feedback)
neotex get <id> --search-id <search_id>
```

## Configuration

### Server (`neotexd`)

| Variable | Required | Description |
|----------|----------|-------------|
| `NEOTEX_DATABASE_URL` | Yes | PostgreSQL connection string |
| `NEOTEX_OPENAI_API_KEY` | Yes | OpenAI API key for embeddings |
| `NEOTEX_EMBEDDING_WORKERS` | No | Number of embedding workers to run (default: 1) |
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
