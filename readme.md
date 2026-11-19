# Job Application Tracker

![CI](https://github.com/Sammmmx/Job-Application-Tracker/actions/workflows/ci.yml/badge.svg)

I kept losing track of where I'd applied, so I built an API to keep track for me. You save applications, filter them by company, status or date, and update them as things move along (applied, interview, offer).

It's written in Go with PostgreSQL. It's also my first real backend project, so I built it the way a production service would be built having separate layers, migrations, three kinds of tests, CI, and Docker.

## What it does

- Create, list, view, replace (`PUT`), partially update (`PATCH`) and delete applications
- Filter by company, role, status, job type, or a date range, with pagination
- Check the input and answer with a clear JSON error instead of crashing
- Fill in sensible defaults: `status` becomes `Applied` and `applied_at` becomes today if you leave them out
- Shut down cleanly: on Ctrl+C or `docker stop`, requests already running are allowed to finish

## Built with

Go (standard library router, so Go 1.22 or newer), PostgreSQL 16, golang-migrate for the schema, GitHub Actions for CI, and Docker Compose to run everything.

## How the code is organised

A request goes through three layers, and each one only talks to the next -

```
middleware -> handler -> service -> repository -> PostgreSQL
```

- **Middleware** (`internal/middleware`): logs every request and turns a panic into a clean 500.
- **Handler** (`handler.go`): reads the HTTP request, calls the service, and turns the result or error into a response.
- **Service** (`service.go`): cleans up the input, applies defaults, and holds the rules.
- **Repository** (`repository.go`): the only place that writes SQL.

Some choices I made on purpose:

- The service depends on a small `Store` interface, not on the repository itself. That is what lets me test the service and handlers against an in-memory fake with no database.
- Errors are wrapped and checked with `errors.Is`, so the handler can turn "not found" into a 404 and "invalid" into a 400. Anything unexpected is logged on the server and the client only sees a generic 500.
- Every query uses placeholders (`$1`, `$2`, ...), so user input never gets glued into SQL. Search text is also escaped, so searching for `100%` finds "100%" instead of matching everything.
- `PATCH` loads the record, applies only the fields you sent, and then runs the same checks as any other write. A small `Nullable` type tells "field left out" apart from `"interview_at": null`, which means "clear it".
- `PUT` insists on a complete record. A forgotten `applied_at` is an error, because quietly filling in today's date would overwrite the real one.
- `interview_at` and `created_at` are `timestamptz`, so the real moment is stored even if you send a time with an offset like `+05:30`.
- `main.go` only wires things together. All the work is in `run()` so the deferred database close actually runs on shutdown (`log.Fatal` would skip it).

```
.
├── cmd/api/                 # main.go: wiring, graceful shutdown
├── internal/
│   ├── application/         # handler, service, repository, models, tests
│   ├── config/              # reads environment variables in one place
│   ├── database/            # Postgres connection and pool settings
│   └── middleware/          # request logging, panic recovery
├── migrations/              # versioned SQL files
├── .github/workflows/       # CI
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

This starts Postgres, applies the migrations, then starts the API at `http://localhost:8080`. Postgres is also exposed on port 5433 if you want to look around with `psql`.

```bash
docker compose down        # stop, keep the data
docker compose down -v     # stop and wipe the database
docker compose logs api    # see the API logs
```

### Without Docker

You need Go, PostgreSQL, and the [migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

1. Create a database, for example `jobtracker`.
2. Copy `.env.example` to `.env` and put your password in `DATABASE_URL`.
3. Apply the migrations:

   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```

4. Start the server:

   ```bash
   go run ./cmd/api
   ```

### Settings

| Variable | Default | What it is |
|---|---|---|
| `DATABASE_URL` | none, required | Postgres connection string |
| `PORT` | `8080` | Port to listen on |

The program stops at startup with a clear message if one of them is missing or wrong.

## Using the API

| Method | Endpoint | What it does | Success |
|---|---|---|---|
| GET | `/applications` | List, with filters and pagination | 200 |
| POST | `/applications` | Create one | 201 |
| GET | `/applications/{id}` | Get one | 200 |
| PUT | `/applications/{id}` | Replace one (send every field) | 200 |
| PATCH | `/applications/{id}` | Change only the fields you send | 200 |
| DELETE | `/applications/{id}` | Delete one | 204 |
| GET | `/health` | Checks the server and the database | 200 |

An application looks like this:

```json
{
  "id": 1,
  "company": "Acme",
  "role": "Backend Developer",
  "job_url": "https://acme.example/jobs/1",
  "status": "Interview",
  "job_type": "Internship",
  "applied_at": "2026-11-01T00:00:00Z",
  "interview_at": "2026-11-10T09:00:00Z",
  "notes": "Referred by a friend",
  "created_at": "2026-11-01T10:15:30Z"
}
```

### What gets accepted

| Field | Rule |
|---|---|
| `company`, `role` | Required, up to 150 characters, surrounding spaces are trimmed |
| `status` | One of `Applied`, `Interview`, `Offer`, `Rejected`, `Withdrawn`. Any capitalisation works and is saved in the standard form. Defaults to `Applied`. |
| `job_type` | Optional. One of `Full-time`, `Part-time`, `Internship`, `Contract`. Any capitalisation works. |
| `job_url` | Optional. Must be a valid `http` or `https` URL. |
| `applied_at` | RFC 3339, like `2026-11-01T00:00:00Z`. Defaults to today on create. Required on `PUT`. |
| `interview_at` | RFC 3339 or `null`. Can't be earlier than `applied_at`. |
| `notes` | Optional |

Other things worth knowing:

- A plain date like `2026-11-01` in the body is rejected. Use the full RFC 3339 form.
- Unknown fields are rejected, so a typo like `"compnay"` gives a 400 instead of being silently ignored.
- Request bodies are limited to 1 MB (413 above that).
- To change the allowed statuses or job types, edit the two lists at the top of `service.go`.

### PUT or PATCH?

`PUT` replaces the whole record, so send every field. `PATCH` changes only what you send:

```bash
# move an application to the interview stage and set the date, nothing else changes
curl -X PATCH localhost:8080/applications/1 -H "Content-Type: application/json" \
  -d '{"status":"interview","interview_at":"2026-11-10T14:30:00+05:30"}'

# clear the interview date
curl -X PATCH localhost:8080/applications/1 -H "Content-Type: application/json" \
  -d '{"interview_at":null}'
```

### Filtering and pagination

`GET /applications` takes these optional query parameters, and you can combine them:

| Parameter | How it matches |
|---|---|
| `company`, `role` | Contains the text, ignoring case |
| `status`, `job_type` | Exact match, ignoring case |
| `start_date`, `end_date` | `YYYY-MM-DD`, applied on or after / on or before |
| `limit` | Page size, 1 to 100, default 20 |
| `offset` | How many to skip, default 0 |

Results come back newest first, wrapped with the paging info:

```json
{ "items": [ ... ], "total": 42, "limit": 20, "offset": 0 }
```

`total` is the number of matches across all pages, so a client knows when to stop.

### Errors

Every error has the same shape:

```json
{ "error": "invalid application: status must be one of Applied, Interview, Offer, Rejected, Withdrawn" }
```

| Status | When |
|---|---|
| 400 | Bad JSON, unknown field, failed validation, bad ID, bad date, bad `limit` or `offset` |
| 404 | No application with that ID |
| 413 | Request body over 1 MB |
| 500 | Something unexpected broke (details are logged, not sent to you) |

On Windows PowerShell, use `curl.exe` and escape the quotes in the JSON, or just use Postman.

## Tests

```bash
go test ./... -cover
```

There are three kinds:

- **Unit tests** for validation, defaults, `PATCH` and pagination, plus config and middleware
- **Handler tests** that send fake HTTP requests through the real router, with an in-memory store instead of a database
- **Repository tests** that run the real SQL against Postgres: filters, search escaping, pagination, time zones

The repository tests wipe the table, so they use their own database and only run when `TEST_DATABASE_URL` is set. Otherwise they are skipped. Apply the migrations to that database first:

```bash
migrate -path migrations -database "$TEST_DATABASE_URL" up
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/jobtracker_test?sslmode=disable"
go test ./... -cover
```

With the repository tests running, the application package has about 92% coverage.

## CI

Every push to `main` and every pull request runs a GitHub Actions workflow. It checks formatting with `gofmt`, runs `go vet`, builds everything, applies the migrations to a temporary Postgres, and runs all the tests with the race detector.

## Limitations

- `PATCH` reads the record and then writes it back, so two requests changing the same record at the same moment can overwrite each other. Locking the row inside a transaction would fix that.
- There are no users or login, so anyone who can reach the API can see everything.
- The count and the page in a list are two separate queries, so the total can be off by a row if someone adds one in between.
- Pagination uses `offset`, which gets slower on very large tables.
- Plain 404 and 405 responses for unknown routes come from Go's router and aren't in the JSON shape.

## Ideas for next

- [ ] User accounts and login
- [ ] Reminders for follow-ups and interview dates
- [ ] A stats endpoint, such as counts per status
- [ ] A history of status changes
- [ ] OpenAPI documentation

## License

MIT, see [LICENSE](LICENSE).
