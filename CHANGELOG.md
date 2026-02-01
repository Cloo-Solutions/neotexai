# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-02-01

### Fixed

- `neotex init` now uses global config credentials from `neotex auth login`

### Changed

- Skills now installed via external `npx skills add` package
- Removed built-in `neotex skill init` command
- Restructured skills to `{name}/SKILL.md` format for CLI compatibility

## [1.0.0] - 2026-02-01

### Added

- Agent-first knowledge management system
- API server (`neotexd`) with RESTful endpoints
- Client CLI (`neotex`) for knowledge operations
- PostgreSQL + pgvector for semantic search
- S3-compatible asset storage integration
- OpenAI embedding generation (text-embedding-ada-002)
- API key authentication with config cascade
- Multi-tenant organization and project support
- Immutable knowledge versioning
- Hybrid search (metadata filtering + semantic similarity)
- Background job processing for async embeddings
- Auth commands (login, logout, status)
- One-line installer script (`curl | sh`)
- Docker image with health checks
- Cross-platform binaries (Linux, macOS, Windows)

[1.1.0]: https://github.com/cloo-solutions/neotexai/releases/tag/v1.1.0
[1.0.0]: https://github.com/cloo-solutions/neotexai/releases/tag/v1.0.0
