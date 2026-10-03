# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project state

ProStock is an early-stage HTTP API server in Go backed by PostgreSQL via `github.com/jackc/pgx/v5` (`pgxpool`). The directory is not a git repository.

- Module path: `ProStock`, Go 1.27, single `main` package at the repo root.
- `main.go` builds an `http.Server` (port from `PORT` env var, default `8080`) around the handler returned by `newRouter(db)`. `DATABASE_URL` is required; the pool connects lazily, so the server starts even when Postgres is down. SIGTERM triggers graceful shutdown.
- Routes are registered in `newRouter()` on a `http.ServeMux` using method-qualified patterns (`"GET /health"`), so wrong methods get 405 automatically. Add new endpoints there.
- Handlers depend on the small `pinger` interface rather than `*pgxpool.Pool`; tests in `main_test.go` pass a `fakeDB` and call `newRouter(...).ServeHTTP` with `httptest`, so no real database or listener is needed.

## Deployment (Dokploy / Docker Swarm)

- `Dockerfile`: multi-stage build into `distroless/static:nonroot`. There is no shell or curl, so the `HEALTHCHECK` runs `/prostock healthcheck` (handled at the top of `main()`). It is a liveness probe: it accepts 200 or 503, so a database outage doesn't make Swarm restart the API replicas.
- `docker-compose.yml`: swarm stack with `api` + `db` (Postgres 17). `api` joins the external `dokploy-network` so Dokploy's Traefik can route to it; `db` is only on the `internal` overlay. `api` receives `DATABASE_URL` built from the `POSTGRES_*` vars.
- `docker stack deploy` ignores `build:`, so stack mode needs `APP_IMAGE` pointing to a pushed image. `POSTGRES_PASSWORD` is required.

## Endpoints

- `GET /health` → `200 {"status":"ok","database":"up"}`, or `503 {"status":"degraded","database":"down"}` when the 2s DB ping fails (error is logged, not returned).

## Commands

```sh
DATABASE_URL=postgres://user:pass@localhost:5432/db go run .   # start server on :8080 (set PORT to override)
go build ./...            # build
go vet ./...              # static checks
gofmt -l -w .             # format
go test ./...             # run all tests
go test -run TestHealth . # run a single test
```
