# Endpoint test suite

Black-box integration tests that exercise every HTTP endpoint through the real
Gin router, a real PostgreSQL database, and real Redis (the same stack the app
uses). Each test sends an actual HTTP request via `httptest` and asserts on the
status code and JSON body.

## What you need

- The `docker-compose` Postgres (mapped to `localhost:5433`) and Redis
  (`localhost:6379`) running.
- A dedicated test database named `noshirvani_test`:

  ```bash
  docker exec backend-postgres-1 psql -U postgres -c "CREATE DATABASE noshirvani_test"
  ```

## Running

```bash
cd backend
go test ./tests/ -v          # all endpoints
go test ./tests/ -run TestExam   # one group
```

Override the connection if your ports differ:

```bash
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5433/noshirvani_test?sslmode=disable' \
TEST_REDIS_ADDR='localhost:6379' \
go test ./tests/
```

## How it works (`main_test.go`)

- `TestMain` connects, runs `AutoMigrate`, builds the router via
  `router.Setup`, and creates a `JWTService` with the same secrets so tests can
  mint tokens directly.
- `resetDB(t)` truncates all tables; call it at the top of any test that depends
  on a clean slate.
- `do(t, method, path, token, body)` performs a request. **Every request is
  given a unique source IP** so the global in-memory rate limiter (60 req/min
  per IP) never trips during the suite.
- `createUser` / `createStudent` / `createAdmin` seed fixtures and return a
  valid bearer token.
- Auth/OTP tests use a real Redis round-trip; `clearOTPLimits` wipes lingering
  per-phone rate-limit keys so they are deterministic across runs.

`known_behavior_test.go` intentionally asserts today's (questionable) behaviour
for the issues listed in `ENDPOINT_TEST_REPORT.md`; if an issue is fixed, the
matching test will fail and should be updated.
