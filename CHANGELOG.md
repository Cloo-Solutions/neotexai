# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2026-02-02

### Added

- Base64 input support for asset uploads via `--base64` flag
- Stdin input support for assets via `--stdin` flag with optional `--encoding base64`
- `--filename` and `--mime-type` flags for non-file asset uploads
- JSONL streaming batch mode for knowledge via `--format jsonl --stream`
- Progress reporting infrastructure for uploads and downloads (`ProgressFunc`, `progressReader`)
- `UploadReader` and `UploadReaderWithProgress` methods for io.Reader-based uploads
- `DownloadFileWithProgress` method for downloads with progress callbacks
- E2E tests for base64 asset uploads and JSONL batch streaming

### Changed

- Asset add command now accepts `[filepath]` as optional (was required)
- Batch add supports streaming processing for memory-efficient large imports
- Internal refactor: asset uploads use `resolveAssetInput()` for unified input handling

## [1.3.0] - 2026-02-02

### Added

- Virtual filesystem (VFS) tools for MCP-compatible agent retrieval
- `POST /context/open` endpoint for content retrieval with line range and char limit support
- `POST /context/list` endpoint for metadata-only listing with filters
- `neotex context open <id>` CLI command with `--chunk`, `--lines`, `--max-chars` flags
- `neotex context list` CLI command with `--path`, `--type`, `--source`, `--since` filters
- Search results now include `chunk_id` and `chunk_index` for precise chunk retrieval
- Asset open supports `--include-url` flag for presigned download URLs
- 47 new unit tests for VFS functionality
- E2E tests for context/open and context/list endpoints

### Fixed

- Sort stability bug in relevance ranking that caused flaky test results (~10% failure rate)
- Added deterministic tiebreaker (ID-based) to `rankByRelevance` function

### Changed

- Search hybrid aggregation now preserves best-matching chunk identity
- CLI search output shows chunk info when available

## [1.2.0] - 2026-02-02

### Added

- Hybrid search results now include snippets, updated timestamps, and chunk matches for long documents
- Knowledge chunk indexing with pgvector + tsvector indexes for faster semantic and lexical retrieval
- Search logging and feedback endpoint; `search_id` surfaced in API/CLI and `neotex get`/`neotex asset get` accept `--search-id`
- CLI search filters: `--status`, `--path`, `--source`, `--mode`, `--project`, `--exact`, plus inline query filters (`type:`, `status:`, `path:`, `source:`, `mode:`, `project:`)
- Search evaluation metrics for precision@k and nDCG@k with optional category breakdown

### Changed

- Default search now uses hybrid semantic + lexical retrieval across knowledge and assets
- CLI search output shows snippets, updated dates, and search IDs when available
- Skills docs updated to cover new search filters and feedback usage

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

[1.4.0]: https://github.com/cloo-solutions/neotexai/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/cloo-solutions/neotexai/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/cloo-solutions/neotexai/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cloo-solutions/neotexai/releases/tag/v1.1.0
[1.0.0]: https://github.com/cloo-solutions/neotexai/releases/tag/v1.0.0
