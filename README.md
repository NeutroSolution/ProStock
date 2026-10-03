# ProStock

HTTP API in Go backed by PostgreSQL. Deployed to [Dokploy](https://dokploy.com) as a Docker Swarm stack.

## Endpoints

| Method | Path      | Response |
|--------|-----------|----------|
| GET    | `/health` | `200 {"status":"ok","version":"1.2.0","database":"up"}`<br>`503 {"status":"degraded","version":"1.2.0","database":"down"}` if Postgres is unreachable |

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

## Releases

Every push to `main` (except Markdown-only changes) runs `.github/workflows/release.yml`, which:

1. runs `go vet` and `go test`
2. computes the next version from the commit messages since the last `v*` tag
3. builds a `linux/amd64` image and pushes `10091991/prostock:<version>` and `:latest`
4. tags the commit `v<version>`

| Commit message                                   | Bump  | Example         |
|--------------------------------------------------|-------|-----------------|
| `feat!: ...`, `fix!: ...` or `BREAKING CHANGE` in body | major | 1.4.2 → 2.0.0 |
| `feat: ...`                                      | minor | 1.4.2 → 1.5.0   |
| anything else (`fix:`, `chore:`, ...)            | patch | 1.4.2 → 1.4.3   |

To force a bump, run the workflow from **Actions → Release → Run workflow**. The running version is returned by `/health`.

Required GitHub repository settings (**Settings → Secrets and variables → Actions**):

- Variable `DOCKERHUB_USERNAME` = `10091991`
- Secret `DOCKERHUB_TOKEN` = a Docker Hub access token with Read & Write scope

To build manually instead:

```sh
docker buildx build --platform linux/amd64 --build-arg VERSION=0.0.0-local \
  -t 10091991/prostock:0.0.0-local --push .
```

## Deploying to Dokploy

`docker-compose.yml` defines a swarm stack with two services:

- **api**: 2 replicas with start-first rolling updates and automatic rollback. Joins `dokploy-network` so Traefik can route to it.
- **db**: Postgres 17 with a persistent `pgdata` volume, reachable only from `api` and pinned to the manager node.

Steps:

1. Create a **Compose** service of type **Stack** pointing at this repository.
2. In **Environment**, set:
   ```
   APP_IMAGE=10091991/prostock:<version>
   POSTGRES_PASSWORD=<openssl rand -hex 24>
   ```
   `POSTGRES_USER` and `POSTGRES_DB` are optional (default `prostock`). Use a URL-safe password because it is embedded in `DATABASE_URL`.
3. In **Domains**, add your domain for service `api`, port `8080`.
4. Deploy, then check `curl https://<your-domain>/health`.

Notes:

- `docker stack deploy` ignores `build:`, so the image must already be pushed. After a release, set `APP_IMAGE` to the new version and redeploy.
- Postgres only reads `POSTGRES_*` on first start with an empty volume. Changing `POSTGRES_PASSWORD` later requires `ALTER ROLE` inside Postgres.
- Dokploy's built-in backups don't cover a database defined in a stack. Set up `pg_dump` backups separately.
