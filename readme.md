# Job Application Tracker

A REST API for tracking job applications, built with Go and PostgreSQL.

## Features

- Create, read, update, and delete job applications
- Filter by company, role, status, job type, and applied date range
- Input validation with clear error responses
- Layered architecture (handler → service → repository)
- SQL migrations with golang-migrate

## Tech Stack

- Go 1.22+ (uses the standard library `net/http` router)
- PostgreSQL
- [lib/pq](https://github.com/lib/pq) as the database driver
- [golang-migrate](https://github.com/golang-migrate/migrate) for migrations

## Project Structure

```
cmd/api/main.go                      Entry point: config and wiring only
internal/application/model.go        Types
internal/application/repository.go   SQL queries
internal/application/service.go      Validation and business rules
internal/application/handler.go      HTTP handlers and routes
internal/database/postgres.go        Database connection
migrations/                          SQL migration files
```

## Getting Started

### Prerequisites

- Go 1.22 or newer
- PostgreSQL
- [migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

### 1. Clone the repository

```
git clone https://github.com/Sammmmx/job-application-tracker.git
cd job-application-tracker
```

### 2. Configure environment

Create a `.env` file in the project root (it is git-ignored):

```
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/YOUR_DB?sslmode=disable
```

### 3. Run migrations

```
migrate -path migrations -database "$DATABASE_URL" up
```

On Windows PowerShell:

```
$db = "postgres://postgres:YOUR_PASSWORD@localhost:5432/YOUR_DB?sslmode=disable"
migrate -path migrations -database $db up
```

### 4. Start the server

```
go mod tidy
go run ./cmd/api
```

The server listens on `http://localhost:8080`.

## API Reference

| Method | Endpoint                | Description                      |
|--------|-------------------------|----------------------------------|
| GET    | `/applications`         | List applications (with filters) |
| POST   | `/applications`         | Create an application            |
| GET    | `/applications/{id}`    | Get one application              |
| PUT    | `/applications/{id}`    | Update an application            |
| DELETE | `/applications/{id}`    | Delete an application            |

### Application fields

| Field          | Type              | Notes                                          |
|----------------|-------------------|------------------------------------------------|
| `id`           | integer           | Generated                                      |
| `company`      | string            | Required, max 150 characters                   |
| `role`         | string
