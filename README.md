# Go Background Job Queue (VA Automation)

A background job processing service written in Go. Clients submit jobs over a REST API; a pool of workers picks them up by priority, runs them, and records every state change in PostgreSQL so work survives restarts.

## Features

- **Worker pool**: a fixed number of concurrent workers process jobs in parallel.
- **Priority queue**: higher-priority jobs are picked up before lower-priority ones.
- **PostgreSQL persistence**: job state, payloads, and results are stored in the database.
- **Retries**: failed jobs are retried up to a configurable limit. [TODO: describe backoff, if any]
- **Cancellation**: queued or running jobs can be cancelled through the API.
- **Recovery**: jobs interrupted by a crash or restart are recovered on startup. [TODO: say which states are re-queued]
- **Job payloads**: each job carries a JSON payload for its handler.
- **Filtering and pagination**: list jobs by status or type, page by page.
- **Graceful shutdown**: on SIGINT/SIGTERM the server stops accepting work and lets in-flight jobs finish. [TODO: state the timeout]

## Tech stack

Go, PostgreSQL, Docker / Docker Compose, REST API.

## Project structure

```
cmd/server/     application entry point
interval/       application code (handlers, queue, workers, storage)
migrations/     SQL migrations for the PostgreSQL schema
data/           [TODO: what lives here]
docker-compose.yml
```

[TODO: one line per sub-package of `interval/`, e.g. domain, queue, worker, storage.]

## Getting started

### With Docker

```bash
docker compose up --build
```

The API listens on `http://localhost:8080`.

### Locally

Requirements: Go [TODO: version from go.mod] and a running PostgreSQL instance.

```bash
# 1. Apply the migrations in ./migrations to your database
# 2. Set the environment variables below
# 3. Start the server
go run ./cmd/server
```

| Variable | Description | Default |
|----------|-------------|---------|
| [TODO: DATABASE_URL] | [TODO] | [TODO] |
| [TODO: worker count] | [TODO] | [TODO] |

## Job lifecycle

[TODO: list your real statuses, for example pending → running → completed / failed / cancelled, and what triggers each transition.]

## Job types

| Type | Purpose |
|------|---------|
| `send_email` | Send an email |
| `generate_report` | Generate a report |
| `process_file` | Process an uploaded file |
| `failing_job` | Always fails; used to exercise retries |

Google integration jobs (Calendar and Sheets: `create_calendar_event`, `list_calendar_events`, `read_sheet`, `append_sheet_rows`, `update_sheet_values`) [TODO: keep this line only if that code is pushed to this repo].

## API

[TODO: replace with your real routes. One row per endpoint.]

| Method | Path | Description |
|--------|------|-------------|
| [TODO] | [TODO] | Create a job |
| [TODO] | [TODO] | List jobs (filter by status/type, paginate) |
| [TODO] | [TODO] | Get one job |
| [TODO] | [TODO] | Cancel a job |

Example request:

```bash
# [TODO: a real curl command that creates a job, with the real route and payload]
```

## Testing

```bash
go test ./...
```

## Roadmap

- `research_web`: a web-crawler job with SSRF and DNS-rebinding protection (in progress).
