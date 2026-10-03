# ProStock

HTTP API in Go backed by PostgreSQL. Deployed to [Dokploy](https://dokploy.com) as a Docker Swarm stack.

## Endpoints

| Method | Path      | Response |
|--------|-----------|----------|
| GET    | `/health` | `200 {"status":"ok","database":"up"}`<br>`503 {"status":"degraded","database":"down"}` if Postgres is unreachable |

## Configuration

| Variable       | Required | Default | Description |
|----------------|----------|---------|-------------|
| `DATABASE_URL` | yes      | —       | Postgres connection string, e.g. `postgres://user:pass@localhost:5432/db?sslmode=disable` |
| `PORT`         | no       | `8080`  | HTTP listen port |

## Local development

Requires Go 1.27+ and a running PostgreSQL.

1. Create the database (as a Postgres superuser):
   ```sql
   CREATE ROLE prostock WITH LOGIN PASSWORD '<password>';
   CREATE DATABASE prostock OWNER prostock;
   ```
2. Create a `.env` file (git-ignored):
   ```
   DATABASE_URL=postgres://prostock:<password>@localhost:5432/prostock?sslmode=disable
   ```
3. Run:
   ```sh
   set -a; source .env; set +a; go run .
   curl -i localhost:8080/health
   ```
   In GoLand, add `DATABASE_URL` to the run configuration's **Environment** field instead.

## Tests

```sh
go test ./...
go test -run TestHealth .   # single test
```

Tests use a fake database and don't need Postgres.

## Docker image

The image is a static binary on `distroless/static:nonroot`. Build for `linux/amd64` (the Dokploy server), even from an Apple Silicon Mac:

```sh
docker login -u 10091991
docker buildx build --platform linux/amd64 \
  -t 10091991/prostock:latest \
  -t 10091991/prostock:v0.1.0 \
  --push .
```

## Deploying to Dokploy

`docker-compose.yml` defines a swarm stack with two services:

- **api**: 2 replicas with start-first rolling updates and automatic rollback. Joins `dokploy-network` so Traefik can route to it.
- **db**: Postgres 17 with a persistent `pgdata` volume, reachable only from `api` and pinned to the manager node.

Steps:

1. Create a **Compose** service of type **Stack** pointing at this repository.
2. In **Environment**, set:
   ```
   APP_IMAGE=10091991/prostock:v0.1.0
   POSTGRES_PASSWORD=<openssl rand -hex 24>
   ```
   `POSTGRES_USER` and `POSTGRES_DB` are optional (default `prostock`). Use a URL-safe password because it is embedded in `DATABASE_URL`.
3. In **Domains**, add your domain for service `api`, port `8080`.
4. Deploy, then check `curl https://<your-domain>/health`.

Notes:

- `docker stack deploy` ignores `build:`, so push the image before deploying, and bump the tag in `APP_IMAGE` for each release.
- Postgres only reads `POSTGRES_*` on first start with an empty volume. Changing `POSTGRES_PASSWORD` later requires `ALTER ROLE` inside Postgres.
- Dokploy's built-in backups don't cover a database defined in a stack. Set up `pg_dump` backups separately.
