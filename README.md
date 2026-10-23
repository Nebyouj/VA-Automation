# Go Background Job Queue (VA Automation)

A background job processing system built with Go, designed to execute asynchronous tasks reliably, manage job priorities, and persist job state using PostgreSQL.

The project provides a REST API for creating, managing, and monitoring background jobs, with a worker pool responsible for processing queued tasks.

## Overview

Background jobs allow applications to perform time-consuming operations without blocking their main request-handling flow. This project explores how to build a job processing system that manages execution, handles failures, and maintains job state.

The system combines a Go-based job processing engine with PostgreSQL persistence and a REST API for interacting with jobs.

## Features

- **Worker pool:** Processes multiple jobs concurrently.
- **Priority queue:** Supports prioritizing jobs for execution.
- **PostgreSQL persistence:** Stores job information and state in a database.
- **Automatic retries:** Retries jobs when execution fails.
- **Job cancellation:** Supports cancelling jobs.
- **Job recovery:** Supports recovering jobs after interruptions.
- **Job payloads:** Allows jobs to carry data required for processing.
- **Filtering and pagination:** Makes it easier to query jobs through the API.
- **Graceful shutdown:** Supports controlled shutdown of the application.

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Backend implementation and job processing |
| PostgreSQL | Persistent job storage |
| Docker | Containerization and development environment |
| REST API | Job creation, management, and monitoring |

## Architecture

At a high level, the system consists of an API layer, job management logic, a queue, workers, and persistent storage.

```text
                Client
                  |
                  v
              REST API
                  |
                  v
            Job Management
                  |
                  v
             Job Queue
                  |
                  v
             Worker Pool
                  |
                  v
           Job Execution

        PostgreSQL
        Persistent job state
```

The API provides an interface for submitting and managing jobs. The processing system schedules queued work for workers, while PostgreSQL stores job information and supports persistence and recovery.

## Getting Started

### Prerequisites

Make sure you have the following installed:

- Go — [Installation guide](https://go.dev/doc/install)
- PostgreSQL
- Docker and Docker Compose, if using the containerized setup

### 1. Clone the repository

```bash
git clone <YOUR_GITHUB_REPOSITORY_URL>
cd <YOUR_PROJECT_DIRECTORY>
```

### 2. Configure environment variables

Create a `.env` file in the project root using the variables required by your application.

For example:

```env
# Replace these with the actual variables
# your application reads.

DATABASE_URL=postgres://<USER>:<PASSWORD>@<HOST>:5432/<DATABASE>?sslmode=disable
```

Check your configuration and database initialization code for the exact variable names and required values.

**Important:** Do not commit credentials, secrets, or your local `.env` file to GitHub.

### 3. Start PostgreSQL

If your repository includes a Docker Compose configuration for PostgreSQL, start it with:

```bash
docker compose up -d
```

Otherwise, start your PostgreSQL instance separately and configure the connection string accordingly.

### 4. Run database migrations

Run the migration command or initialization procedure used by your project.

```bash
# Replace with your actual migration command.
```

### 5. Start the application

```bash
go run ./cmd/server
```

If your application uses a different entry point, replace the command above with the appropriate one.

Once the server starts, check the configured address and port to access the REST API.

## API Endpoints

The REST API allows clients to create, retrieve, filter, and manage background jobs.

Replace the placeholders below with the actual routes and HTTP methods implemented in your application.

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/jobs` | Create a job |
| `GET` | `/jobs` | List jobs with filtering and pagination |
| `GET` | `/jobs/{id}` | Retrieve a job by ID |
| `POST` | `/jobs/{id}/cancel` | Cancel a job |
| `POST` | `/jobs/{id}/retry` | Retry a failed job, if supported |

### Example: Creating a job

The following illustrates a generic job creation request. Adapt the fields to match your API's actual request structure.

```http
POST /jobs
Content-Type: application/json
```

```json
{
  "type": "example_task",
  "payload": {
    "message": "Hello from the job queue"
  },
  "priority": 1
}
```

Example response structure:

```json
{
  "id": "<job-id>",
  "status": "queued"
}
```

The request and response above are illustrative. Refer to your actual API handlers and models for the supported fields, status values, and response format.

## Job Lifecycle

A typical job moves through several stages:

1. **Submission:** A client sends a request to create a job.
2. **Queueing:** The job is added to the queue with its associated payload and priority.
3. **Execution:** An available worker picks up the job and executes it.
4. **Completion or failure:** The job's outcome is recorded.
5. **Retry or recovery:** Jobs eligible for another attempt or recovery are handled by the corresponding mechanisms.

The exact lifecycle and status transitions depend on the implementation.

## Testing

Run the Go test suite from the repository root:

```bash
go test ./...
```

To run tests with the race detector:

```bash
go test -race ./...
```

The race detector is particularly useful when testing concurrent job processing.

Tests that require PostgreSQL may need a running database and appropriate environment variables.

## Project Structure

The repository is organized around the components responsible for job management, processing, persistence, and API handling.

Update this example to reflect your actual directory structure:

```text
.
├── cmd/
│   └── server/
├── interval/
│   ├── application/
│   ├── domain/
│   ├── infrastructure/
│   ├── integrations/
│   └── interfaces/
├── tests/
├── go.mod
├── go.sum
└── README.md
```

## Design Goals

This project focuses on exploring:

- Concurrent task execution using Go.
- Queue and worker coordination.
- Persistent job state and recovery.
- Failure handling and retry mechanisms.
- API design for job management.
- Reliable application shutdown.

## Future Improvements

Potential areas for further development include:

- Job scheduling with cron expressions.
- Improved observability through metrics and structured logging.
- Authentication and authorization for the API.
- More comprehensive integration and load testing.
- Deployment documentation and operational monitoring.

These are potential improvements, not claims about existing functionality.

## Author

**Nebyou Damtew**

Computer and Electrical Engineering student interested in backend development, distributed systems, and software engineering.
