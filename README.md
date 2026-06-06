# Marrow

Marrow is a Go-based RSS/Feed polling service that fetches feed entries from configurable sources, stores them in SQLite, and exposes a small HTTP API for latest items and polling state. It supports hot-reloading of source definitions and can optionally route feed requests through RSSHub for non-standard or scraper-backed endpoints.

## Key Features

- Config-driven feed polling using `configs/sources.yml`
- Source-specific poll intervals
- Persistent SQLite storage in `./data/marrow.db`
- HTTP API for latest feed items, source metadata, and polling state
- Server-Sent Events (SSE) endpoint for real-time updates
- Hot-reload `sources.yml` on change using `fsnotify`
- Optional RSSHub support via `RSSHUB_BASE_URL`

## Architecture

The application is structured around a simple pipeline:

1. Load application configuration from environment variables
2. Load feed sources from `sources.yml`
3. Start a poller per source
4. Poll each feed on its own interval
5. Parse feed content using `gofeed`
6. Persist new items to SQLite
7. Publish new items through an internal notification channel
8. Expose data through HTTP endpoints

### Main components

- `cmd/marrow-api/main.go` — application entrypoint
- `internal/config` — app env config + YAML source loading + hot reload watcher
- `internal/feed` — manages pollers and broadcast notifications
- `internal/poller` — source polling, request handling, backoff, deduplication
- `internal/parser` — converts raw feed responses into domain items
- `internal/storage/sqlite` — SQLite repository layer and schema bootstrapping
- `internal/transport/http` — HTTP handlers and router
- `internal/domain` — domain models for feed items, sources, and polling state

## Prerequisites

- Go 1.25+
- `git`
- Optional: Docker / Docker Compose if you want to run RSSHub locally

## Quickstart

### 1. Clone the repository

```bash
git clone https://github.com/Master-Daniels/marrow.git
cd marrow
```

### 2. Configure environment variables

Copy or inspect `sample.env` and set the values you need.

Example:

```bash
export RSSHUB_BASE_URL=http://localhost:1200
export DATABASE_URL=./data/marrow.db
export APP_PORT=8080
export CONFIG_PATH=./configs/
export LOG_LEVEL=debug
```

> Note: `CONFIG_PATH` should include a trailing slash because the application concatenates it with `sources.yml`.

### 3. Configure sources

Edit `configs/sources.yml` to define your feed sources.

Example:

```yaml
sources:
  - id: "Elon Musk Twitter"
    display_name: "Elon Musk Twitter"
    type_hint: "twitter"
    feed_url: "/twitter/user/elon"
    poll_interval: "5m"
```

### 4. Run locally

From the repository root:

```bash
go run ./cmd/marrow-api/main.go
```

The server listens on the port defined by `APP_PORT` (default `8080`).

## Environment Variables

- `RSSHUB_BASE_URL` — optional base URL for RSSHub proxy requests
- `DATABASE_URL` — SQLite database path (not currently used in all code paths, but available for future support)
- `APP_PORT` — HTTP server port (default: `8080`)
- `CONFIG_PATH` — directory containing `sources.yml` (default: `../../configs`)
- `LOG_LEVEL` — logger level (default: `info`)

## HTTP API

The service exposes the following endpoints:

- `GET /api/v1/feed?limit=N`
  - Returns the latest feed items as JSON
  - Default `limit` is `20`
- `GET /api/v1/updates`
  - Server-Sent Events (SSE) stream of newly inserted feed items
- `GET /api/v1/sources/{sourceID}`
  - Returns the persisted source configuration for a given source ID
- `GET /api/v1/sources/polling_state`
  - Returns polling state for all sources
- `GET /api/v1/sources/{sourceID}/polling_state`
  - Returns polling state for a single source

### Example curl

```bash
curl "http://localhost:8080/api/v1/feed?limit=10"
```

```bash
curl "http://localhost:8080/api/v1/sources/polling_state"
```

## Data Model

### Feed items

Feed items are represented by `internal/domain.FeedItem` and contain:

- `id` — deterministic item ID
- `source_id` — source identifier
- `title`
- `description`
- `link`
- `published`
- `type` — one of `tweet`, `article`, `podcast`, `video`, or `generic`
- optional `enclosure`, `duration`, and `thumbnail`

### Sources

Sources are defined in `configs/sources.yml` and persisted in the `sources` table.

### Polling state

The service tracks polling metadata in a `polling_state` table, including:

- `last_successful_poll`
- `consecutive_failures`
- `last_error`
- `last_polled_at`

## Storage

- SQLite database is bootstrapped from `internal/storage/schema.sql`
- Default runtime database path created under `./data/marrow.db`
- `internal/storage/sqlite/repository.go` provides insertion and query methods

## Docker support

A Docker Compose file is included at `deployments/docker-compose.yml` to run the API alongside RSSHub.

```bash
docker compose -f deployments/docker-compose.yml up --build
```

This setup includes:

- `rsshub` service on port `1200`
- `marrow-api` service on port `8080`

## Project Layout

- `cmd/marrow-api` — app entrypoint
- `configs` — feed source definitions
- `internal/config` — config loading + YAML source parsing
- `internal/feed` — poller orchestration + SSE notifications
- `internal/poller` — feed fetch, parsing, backoff, deduplication
- `internal/parser` — feed parsing via `gofeed`
- `internal/storage` — repository interface + SQLite implementation
- `internal/transport/http` — API router and handlers
- `internal/domain` — shared domain models
- `deployments` — Docker Compose and Dockerfile

## Notes

- Sources are hot-reloaded when `configs/sources.yml` changes
- Feed polling runs in separate goroutines for each source
- Duplicate entries are deduplicated before persistence
- Current HTTP API implementation is minimal and intended for extension

## Contributing

If you want to extend Marrow, a few good next steps are:

- add more HTTP endpoints for filtering and pagination
- improve source state and error handling
- add tests for parser, poller, and storage layers
- support additional feed formats and auth headers

---

Built with Go and SQLite to provide a lightweight feed ingestion and delivery service.