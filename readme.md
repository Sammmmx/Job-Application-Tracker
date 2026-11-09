# Job Application Tracker

![CI](https://github.com/Sammmmx/Job-Application-Tracker/actions/workflows/ci.yml/badge.svg)

I kept losing track of where I'd applied, so I built a small API to do it for me. You can save applications, filter them by company, status or date, and update or delete them as things change.

It's written in Go with PostgreSQL. It's also my first real backend project, so I tried to build it the way a production service would be built: separate layers, migrations, tests, CI, and Docker.

## What it does

- Create, list, view, update and delete job applications
- Filter by company, role, status, job type, or a date range
- Reject bad input with a clear 400 message instead of a crash
- Fill in sensible defaults: status becomes `Applied` and the applied date becomes today if you leave them out

## Built with

Go (standard library router, so Go 1.22 or newer), PostgreSQL 16, golang-migrate for the schema, GitHub Actions for CI, and Docker Compose to run everything.

## How the code is organised

A request passes through three layers, and each one only talks to the next:

```
handler  ->  service  ->  repository  ->  PostgreSQL
```

- **Handler** (`handler.go`): reads the HTTP request, calls the service, and turns the result or error into a response.
- **Service** (`service.go`): applies defaults and validation. This is where the rules live.
- **Repository** (`repository.go`): the only place that writes SQL.

A few choices I made on purpose:

- The service depends on a small `Store` interface, not on the repository directly. That's what lets me test the service and handlers with an in-memory fake, with no database running.
- Errors are wrapped and checked with `errors.Is`, so the handler can turn "not found" into a 404 and "invalid" into a 400. Anything unexpected is logged on the server and the client only sees a generic 500.
- Every query uses placeholders (`$1`, `$2`, ...) so user input never gets glued into SQL.

```
.
├── cmd/api/              # main.go: wiring and server setup
├── internal/
│   ├── application/      # handler, service, repository, models, tests
│   ├── database/         # Postgres connection and pool settings
│   └── middleware/       # empty for now (logging and recovery are planned)
├── migrations/           # versioned SQL files
├── .github/workflows/    # CI
├── Dockerfile
└── docker-compose.yml
```

## Running it

### With Docker (easiest)

You need Docker Desktop. Then:

```bash
git clone https://github.com/Sammmmx/Job-Application-Tracker.git
cd Job-Application-Tracker
docker compose up --build
```

This starts Postgres, applies the migrations, and then starts the API. Once you see `Server listening on :8080`, it's ready at `http://localhost:8080`. Postgres is also exposed on port 5433 if you want to poke around with `psql`.

```bash
docker compose down        # stop, keep the data
docker compose down -v     # stop and wipe the database
docker compose logs api    # see the API logs
```

### Without Docker

You'll need Go, PostgreSQL, and the [migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

1. Create a database, for example `jobtracker`.
2. Make a `.env` file in the project root:

   ```
   DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/jobtracker?sslmode=disable
   ```

3. Apply the migrations:

   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```

4. Start the server:

   ```bash
   go run ./cmd/api
   ```

## Using the API

| Method | Endpoint             | What it does                      | Success |
|--------|----------------------|-----------------------------------|---------|
| GET    | `/applications`      | List applications (filters below) | 200     |
| POST   | `/applications`      | Create one                        | 201     |
| GET    | `/applications/{id}` | Get one                           | 200     |
| PUT    | `/applications/{id}` | Replace one                       | 200     |
| DELETE | `/applications/{id}` | Delete one                        | 204     |

An application looks like this:

```json
{
  "id": 1,
  "company": "Acme",
  "role": "Backend Developer",
  "job_url": "https://example.com/job/123",
  "status": "Applied",
  "job_type": "Full-time",
  "applied_at": "2026-11-01T00:00:00Z",
  "interview_at": null,
  "notes": "Referred by a friend",
  "created_at": "2026-11-01T10:15:30Z"
}
```

Only `company` and `role` are required, and both can be up to 150 characters. `status` and `job_type` can be up to 30. If you leave out `applied_at` it becomes today, and if you set `interview_at` it can't be earlier than `applied_at`.

A few things worth knowing:

- Dates must be RFC 3339, like `2026-11-01T00:00:00Z`. A plain `2026-11-01` is rejected.
- Unknown fields are rejected, so a typo like `"compnay"` gives you a 400 instead of being silently ignored.
- `PUT` replaces the whole record, so send every field, not just the one you're changing.

### Filtering

`GET /applications` takes these optional query parameters, and you can combine them:

| Parameter    | How it matches                          |
|--------------|-----------------------------------------|
| `company`    | Contains the text, ignoring case        |
| `role`       | Contains the text, ignoring case        |
| `status`     | Exact match, ignoring case              |
| `job_type`   | Exact match, ignoring case              |
| `start_date` | `YYYY-MM-DD`, applied on or after       |
| `end_date`   | `YYYY-MM-DD`, applied on or before      |

Results come back newest first.

### Examples

```bash
# create
curl -X POST http://localhost:8080/applications \
  -H "Content-Type: application/json" \
  -d '{"company":"Acme","role":"Backend Developer","job_type":"Full-time"}'

# list with filters
curl "http://localhost:8080/applications?status=applied&company=acme"

# get, update, delete
curl http://localhost:8080/applications/1
curl -X PUT http://localhost:8080/applications/1 \
  -H "Content-Type: application/json" \
  -d '{"company":"Acme","role":"Backend Developer","status":"Interview"}'
curl -X DELETE http://localhost:8080/applications/1
```

On Windows PowerShell, use `curl.exe` (and escape the quotes in the JSON), or just use Postman.

### Errors

| Status | When                                                                 |
|--------|----------------------------------------------------------------------|
| 400    | Bad JSON, unknown field, failed validation, bad ID, or bad date      |
| 404    | No application with that ID                                          |
| 500    | Something unexpected broke (details are logged, not sent to you)     |

## Tests

```bash
go test ./... -cover
```

There are three kinds of tests:

- **Unit tests** for validation and defaults
- **Handler tests** that send fake HTTP requests through the real router, with an in-memory store instead of a database
- **Repository tests** that run the real SQL against Postgres

The repository tests wipe the table, so they use their own database and only run when `TEST_DATABASE_URL` is set. Otherwise they're skipped:

```bash
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/jobtracker_test?sslmode=disable"
go test ./internal/application -run Repo -v
```

## CI

Every push to `main` and every pull request runs a GitHub Actions workflow. It checks formatting with `gofmt`, runs `go vet`, builds everything, applies the migrations to a temporary Postgres, and runs all the tests with the race detector.

## What I want to add next

- [ ] Graceful shutdown
- [ ] Request logging and panic recovery
- [ ] JSON error responses and a request body size limit
- [ ] A fixed list of allowed `status` and `job_type` values
- [ ] Pagination
- [ ] PATCH for partial updates
- [ ] User accounts and login

## License

MIT
