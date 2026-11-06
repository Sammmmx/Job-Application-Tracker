# Job Application Tracker

A backend REST API, written in Go, for keeping track of the jobs you apply to. Create applications, update their status as you move through the hiring process, and keep everything stored in PostgreSQL.

> Built as a learning project to practice backend fundamentals: layered architecture, database migrations, testing, and CI.

## Features

- Create, read, update, and delete job applications
- Persistent storage in PostgreSQL
- Versioned database migrations
- Layered architecture (handler → service → repository)
- Unit tests for the handler and service layers, using a fake repository
- Continuous integration with GitHub Actions

## Tech Stack

| Area | Tool |
|------|------|
| Language | Go |
| Database | PostgreSQL |
| Migrations | SQL migration files (`migrations/`) |
| Testing | Go `testing` package, with coverage reports |
| CI | GitHub Actions |

## Project Structure

```
.
├── .github/workflows/
│   └── ci.yml                  # CI pipeline
├── cmd/api/
│   └── main.go                 # Application entry point
├── internal/
│   ├── application/
│   │   ├── handler.go          # HTTP handlers (request/response)
│   │   ├── handler_test.go
│   │   ├── service.go          # Business logic
│   │   ├── service_test.go
│   │   ├── repository.go       # Data access layer
│   │   ├── model.go            # Domain models
│   │   └── fake_test.go        # Fake repository used in tests
│   └── database/
│       └── postgres.go         # PostgreSQL connection setup
├── migrations/
│   ├── 000001_create_job_applications.(up|down).sql
│   └── 000002_make_text_columns_not_null.(up|down).sql
├── go.mod
├── go.sum
└── README.md
```

### How the layers fit together

```
HTTP request → Handler → Service → Repository → PostgreSQL
```

- **Handler**: parses requests, validates input, writes responses.
- **Service**: holds the business rules.
- **Repository**: the only layer that talks to the database.

Keeping these separate makes each layer easy to test on its own. The service tests use a fake repository, so they don't need a real database.

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.21 or newer (check the version in `go.mod`)
- [PostgreSQL](https://www.postgresql.org/download/)
- A migration tool such as [golang-migrate](https://github.com/golang-migrate/migrate)

### 1. Clone the repository

```bash
git clone https://github.com/<your-username>/<your-repo-name>.git
cd <your-repo-name>
```

### 2. Set up the database

Create a database:

```bash
createdb job_tracker
```

### 3. Configure environment variables

```bash
# Example, adjust to match what main.go reads
export DATABASE_URL="postgres://user:password@localhost:5432/job_tracker?sslmode=disable"
export PORT=8080
```

### 4. Run migrations

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

### 5. Run the server

```bash
go run ./cmd/api
```

The API will be available at `http://localhost:8080`.

## API Endpoints

> Update this table to match the routes registered in `handler.go`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/applications` | Create a new job application |
| `GET` | `/applications` | List all job applications |
| `GET` | `/applications/{id}` | Get a single application |
| `PUT` | `/applications/{id}` | Update an application |
| `DELETE` | `/applications/{id}` | Delete an application |

### Example request

```bash
curl -X POST http://localhost:8080/applications \
  -H "Content-Type: application/json" \
  -d '{
    "company": "Acme Corp",
    "position": "Backend Engineer",
    "status": "applied"
  }'
```

### Example response

```json
{
  "id": 1,
  "company": "Acme Corp",
  "position": "Backend Engineer",
  "status": "applied"
}
```

## Running Tests

Run all tests:

```bash
go test ./...
```

Run tests with a coverage report:

```bash
go test ./... -coverprofile=cover
go tool cover -html=cover
```

## Continuous Integration

Every push and pull request triggers the GitHub Actions workflow in `.github/workflows/ci.yml`, which builds the project and runs the test suite.

## Database Migrations

Migrations live in `migrations/` and are applied in order:

| Migration | Purpose |
|-----------|---------|
| `000001_create_job_applications` | Creates the `job_applications` table |
| `000002_make_text_columns_not_null` | Adds `NOT NULL` constraints to text columns |

To roll back the last migration:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

## What I Learned

- Structuring a Go project with `cmd/` and `internal/`
- Separating handler, service, and repository layers
- Writing database migrations
- Testing with fakes instead of a live database
- Setting up a CI pipeline

## Future Improvements

- [ ] User authentication
- [ ] Filtering and searching applications by status, company, or date
- [ ] Reminders for follow-ups and interview dates
- [ ] Docker and docker-compose setup
- [ ] API documentation with OpenAPI/Swagger

## License

MIT License