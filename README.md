# TaskFlow

A task management backend with authentication, projects, and tasks — built in Go with PostgreSQL.

---

## 1. Overview

TaskFlow lets users register, log in, create projects, and manage tasks within those projects. Tasks can be assigned to users, filtered by status and assignee, and tracked through a `todo → in_progress → done` lifecycle.

**Tech stack:**

| Layer | Technology | Why |
|---|---|---|
| Language | Go 1.22 | Required by assignment; strong stdlib, great concurrency primitives |
| Router | chi | `net/http`-compatible, easy-to-write middleware, follows idomatic Go and better control |
| Database driver | pgx/v5 + sqlx | Maintained Postgres driver; sqlx adds struct scanning without ORM requirement |
| Migrations | golang-migrate | Supports up/down, embeds into binary via `iofs`, runs automatically on startup |
| Auth | bcrypt + golang-jwt/v5 | bcrypt cost 12; HS256 JWT with 24h expiry |
| Logging | slog (stdlib) | Structured JSON logging, zero external dependencies (Go 1.21+) |
| Tests | testcontainers-go | Integration tests hit a real Postgres container — no mocking |

---

## 2. Architecture Decisions

### 3-Layer Clean Architecture

```
Handler → Service → Repository → PostgreSQL
```

- **Handler**: HTTP concerns only — parse body, validate input, call service, write JSON.
- **Service**: Business logic and authorization checks. Follows rules like "only the project owner can delete."
- **Repository**: Raw SQL via sqlx. One function per query.

### Why raw SQL instead of an ORM?

The assignment explicitly prohibits ORM magic. Raw SQL also means every query is visible and optimizable. The repository layer is the single source of truth for all database access.

### Why chi over Gin/Echo?

chi handlers are standard `http.HandlerFunc` — no framework-specific types come into the codebase. Middleware composes using idiomatic Go standard patterns.

### PostgreSQL enums for status/priority

`task_status` and `task_priority` are PostgreSQL `ENUM` types. This enforces validity at the database level — an application bug can never insert `"urgent"` where `"high"` is expected. The tradeoff: adding new values requires a migration.

### `created_by` column on tasks

Tasks store a created_by field to track who created them. This is required to enforce the "project owner or task creator" delete rule — without it, there's no way to verify the caller is the original creator. It's not in the original schema spec but is a necessary addition to implement the authorization correctly.

### Migration embedding

All SQL files are embedded into the binary (`//go:embed *.sql`). The container is a single static binary — no external migration tooling, no file system paths to configure.

### Seed as Go code

Seeding runs bcrypt at startup to generate a correct hash, then inserts with `ON CONFLICT DO NOTHING`. Idempotent and always verifiably correct — no pre-generated hash to trust.

---

## 3. Running Locally

**Requirements:** Docker and Docker Compose only.

```bash
git clone https://github.com/dukedhal/taskflow-duke
cd taskflow-duke

# Optional: customize env (works with defaults as-is)
cp .env.example .env

# Start everything: Postgres + API
docker compose up --build
```

The API is available at `http://localhost:8080`.

On first start:
1. Postgres starts and passes its healthcheck
2. The API binary starts, runs all migrations, then inserts seed data
3. The server begins accepting requests

To stop: `docker compose down`

To reset the database: `docker compose down -v && docker compose up --build`

---

## 4. Running Migrations

Migrations run **automatically on every container start**. Nothing to run manually.

To run with the migrate CLI against a local Postgres:
```bash
migrate -path ./backend/migrations \
  -database "postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable" \
  up
```

---

## 5. Running Tests

The test suite uses **testcontainers-go** — each test spins up a real `postgres:16-alpine` Docker container, runs all migrations against it, and tears it down when done. No mocking, no manual DB setup required.

**Requirements:** Docker must be running on your machine.

```bash
cd backend
go test ./tests/... -v
```

This runs all 3 test files covering auth, projects, and tasks (11 tests total). The first run may be slower as Docker pulls the Postgres image.

To run a specific test:
```bash
go test ./tests/... -v -run TestAuth
go test ./tests/... -v -run TestProjects
go test ./tests/... -v -run TestTasks
```

---

## 6. Dummy Credentials

Seed data is inserted automatically on startup (`RUN_SEED=true` by default).

```
Email:    test@example.com
Password: password123
```

The seed also creates:
- 1 project: **"Demo Project"**
- 3 tasks: one `done`, one `in_progress`, one `todo`

---

## 6. API Reference

All endpoints return `Content-Type: application/json`.
Protected endpoints require `Authorization: Bearer <token>`.

### Auth

---

#### `POST /auth/register`

```bash
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe","email":"jane@example.com","password":"securepassword"}'
```

**201 Created**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "created_at": "2026-04-13T10:00:00Z"
  }
}
```

**400 Validation failed**
```json
{ "error": "validation failed", "fields": { "password": "must be at least 8 characters" } }
```

**409 Email already registered**
```json
{ "error": "email already registered" }
```

---

#### `POST /auth/login`

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

**200 OK**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Test User",
    "email": "test@example.com",
    "created_at": "2026-04-13T09:00:00Z"
  }
}
```

**401 Wrong credentials**
```json
{ "error": "invalid credentials" }
```

---

### Users

#### `GET /users`
> Requires: `Authorization: Bearer <token>`

```bash
curl -s http://localhost:8080/users \
  -H "Authorization: Bearer $TOKEN"
```

**200 OK**
```json
{
  "users": [
    { "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "name": "Test User", "email": "test@example.com", "created_at": "2026-04-13T09:00:00Z" },
    { "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901", "name": "Jane Doe",  "email": "jane@example.com",  "created_at": "2026-04-13T10:00:00Z" }
  ]
}
```

---

### Projects

#### `POST /projects`
> Requires: `Authorization: Bearer <token>`

```bash
curl -s -X POST http://localhost:8080/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Website Redesign","description":"Q2 initiative"}'
```

**201 Created**
```json
{
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "name": "Website Redesign",
  "description": "Q2 initiative",
  "owner_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-04-13T10:05:00Z"
}
```

---

#### `GET /projects?page=1&limit=20`
> Returns projects the caller owns or has tasks assigned in.

```bash
curl -s "http://localhost:8080/projects?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

**200 OK**
```json
{
  "projects": [
    {
      "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "name": "Website Redesign",
      "description": "Q2 initiative",
      "owner_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "created_at": "2026-04-13T10:05:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

---

#### `GET /projects/:id`
> Returns the project and all its tasks.

```bash
curl -s http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012 \
  -H "Authorization: Bearer $TOKEN"
```

**200 OK**
```json
{
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "name": "Website Redesign",
  "description": "Q2 initiative",
  "owner_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-04-13T10:05:00Z",
  "tasks": [
    {
      "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
      "project_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "title": "Design homepage",
      "description": null,
      "status": "todo",
      "priority": "high",
      "assignee_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "due_date": "2026-05-01",
      "created_by": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "created_at": "2026-04-13T10:10:00Z"
    }
  ]
}
```

---

#### `PATCH /projects/:id`
> Partial update — owner only. Any subset of fields accepted.

```bash
curl -s -X PATCH http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Website Redesign v2"}'
```

**200 OK** — returns updated project object.

**403 Forbidden** (not the owner)
```json
{ "error": "forbidden" }
```

---

#### `DELETE /projects/:id`
> Owner only. Cascades to all tasks.

```bash
curl -s -X DELETE http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012 \
  -H "Authorization: Bearer $TOKEN"
```

**204 No Content** — empty body on success.

---

#### `GET /projects/:id/stats`

```bash
curl -s http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012/stats \
  -H "Authorization: Bearer $TOKEN"
```

**200 OK**
```json
{
  "by_status": {
    "todo": 2,
    "in_progress": 1,
    "done": 3
  },
  "by_assignee": [
    { "user_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901", "name": "Jane Doe", "count": 4 },
    { "user_id": null, "name": "Unassigned", "count": 2 }
  ],
  "total": 6
}
```

---

### Tasks

#### `POST /projects/:id/tasks`

```bash
curl -s -X POST http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Design homepage",
    "priority": "high",
    "assignee_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "due_date": "2026-05-01"
  }'
```

**201 Created**
```json
{
  "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
  "project_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "title": "Design homepage",
  "description": null,
  "status": "todo",
  "priority": "high",
  "assignee_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "due_date": "2026-05-01",
  "created_by": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-04-13T10:10:00Z"
}
```

---

#### `GET /projects/:id/tasks`
> Filter by `status` and/or `assignee`. Supports pagination.

```bash
curl -s "http://localhost:8080/projects/c3d4e5f6-a7b8-9012-cdef-123456789012/tasks?status=todo&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

**200 OK**
```json
{
  "tasks": [
    {
      "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
      "project_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "title": "Design homepage",
      "description": null,
      "status": "todo",
      "priority": "high",
      "assignee_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "due_date": "2026-05-01",
      "created_by": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "created_at": "2026-04-13T10:10:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

---

#### `PATCH /tasks/:id`
> All fields optional. Task creator or project owner only.

```bash
curl -s -X PATCH http://localhost:8080/tasks/d4e5f6a7-b8c9-0123-defa-234567890123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress","priority":"low"}'
```

**200 OK** — returns full updated task object.

---

#### `DELETE /tasks/:id`
> Task creator or project owner only.

```bash
curl -s -X DELETE http://localhost:8080/tasks/d4e5f6a7-b8c9-0123-defa-234567890123 \
  -H "Authorization: Bearer $TOKEN"
```

**204 No Content** — empty body on success.

**403 Forbidden** (neither creator nor project owner)
```json
{ "error": "forbidden" }
```

---

### Error Response Format

```json
{ "error": "not found" }
{ "error": "validation failed", "fields": { "title": "is required" } }
```

HTTP status mapping:
| Code | Meaning |
|---|---|
| 400 | Validation failure / bad input |
| 401 | Missing, invalid, or expired JWT |
| 403 | Valid JWT but not authorized for this action |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate email) |
| 500 | Unexpected server error |

---

## 7. What You'd Do With More Time

**Security:**
- Rate limiting on auth endpoints (prevent brute force)
- Refresh token rotation (current: single 24h access token, no revocation)
- CORS configuration (for frontend integration)

**Observability:**
- Request correlation ID propagated through all log lines
- `/health` and `/ready` endpoints for orchestrators
- Structured metrics, logs, error reporting (Prometheus, Grafana or Loki)

**Data model:**
- Soft deletes (`deleted_at`) for audit trail (currently hard-delete supported for low complexity)
- Task activity log (who changed what and when)
- `GET /users` endpoint for assignee picker
- Bulk status updates, sorting option on list (by priority, due date)

**Infrastructure:**
- API versioning (`/v1/` prefix)
- GitHub Actions CI pipeline (lint, vet, test, build)
- Multi-environment Docker Compose overrides

**Optimizations:**
- Database connection pool can be tuned as per load profile.
- Response caching on read-heavy endpoints like `GET /projects/:id/stats` (short TTL, invalidated on task write)
- Database-level chunking using PostgreSQL optimized CTID for large dataset.

