# hospital-middleware-system

A middleware API that lets hospital staff search and view patient information sourced from each hospital's own Hospital Information System (HIS). Staff authenticate against a specific hospital and can only ever search patients belonging to that hospital.

## How it works

- **Staff accounts are scoped to a single hospital.** `username` is unique per hospital, not globally, so the same username can exist under different hospitals as different accounts. Login issues a JWT that embeds the staff's `hospital_id` — every later request is scoped using that claim, never a client-supplied value.
- **Patient data is cached locally, refreshed from the HIS on demand.** The `patients` table mirrors each hospital's own records, scoped per hospital (`UNIQUE(hospital_id, national_id)`, not a global unique — the same person can be a patient at two different hospitals with two independent records). When a search includes `national_id` or `passport_id` (the only identifier Hospital A's HIS route actually accepts), the middleware calls that hospital's HIS, upserts the response locally, and then searches — so results always reflect both the fresh HIS data and every other filter the client asked for.
- **The HIS is an enhancement, not a dependency.** If the upstream HIS is unreachable or errors, the search still returns whatever's already cached locally instead of failing the request.
- **Onboarding another hospital is a data change, not a code change.** `hospitals` is a real table (`code`, `name`, `api_base_url`); a new hospital's HIS adapter just needs to be plugged in behind the same `HISClient` interface used for Hospital A.

## API

| Method | Route | Auth | Description |
|---|---|---|---|
| POST | `/staff/create` | — | Create a staff login scoped to a hospital (`username`, `password`, `hospital`) |
| POST | `/staff/login` | — | Authenticate and receive an access/refresh JWT pair |
| POST | `/patient/search` | Bearer token | Search patients belonging to the logged-in staff's hospital; all filter fields are optional |

Full request/response schemas are served at `/swagger/index.html` once the app is running.

## Tech stack

- **Go** 1.25 - API server, using [go.uber.org/fx](https://github.com/uber-go/fx) for dependency injection
- **[Gin](https://github.com/gin-gonic/gin)** - HTTP routing
- **JWT** ([golang-jwt](https://github.com/golang-jwt/jwt)) - staff authentication, with bcrypt for password hashing
- **Docker** / **docker-compose** - containerized app + database
- **nginx** - reverse proxy in front of the app container
- **PostgreSQL** - primary datastore
- **[gorm](https://gorm.io/)** - ORM; models generated with `gorm.io/gen`
- **[goose](https://github.com/pressly/goose)** - SQL migrations
- Swagger docs via [swaggo](https://github.com/swaggo/swag)

## Project layout

```
core/            business logic per domain (patient, staff) - handler -> service -> repo
infra/           cross-cutting concerns: db, auth (JWT), web (Gin engine + routing), config
infra/db/model/  gorm models generated from the live schema (gen-model) - do not hand-edit
infra/db/migrations/  goose SQL migrations (source of truth for the schema)
cmd/gen-model/   regenerates infra/db/model from the live DB schema
cmd/gen-repo/    regenerates each core/*/repo.go interface from its repo_*.go implementations
```

## Running it

### Docker Compose (recommended)

```bash
cp .env.example .env
# set APP_MIGRATE=true in .env so the app migrates the DB itself on boot
docker compose up --build
```

This starts three services: `nginx` (port 80, proxies to the app), `app` (the Go API), and `postgres`. `.env.example` defaults `APP_MIGRATE` to `false` (assuming migrations are applied out-of-band via `task db-migrate-up`); for Compose, setting it to `true` is simpler since it avoids needing host-side `goose` access to the container's Postgres at all.

> **Port 5432 gotcha:** if you have a local/native Postgres already listening on `5432` (e.g. a Homebrew service), it will silently intercept connections to `localhost:5432` from your host machine before Docker's proxy does — `psql`, `goose`, and `task gen-model` run from the host will all talk to the wrong database. This doesn't affect the app itself (it connects via the internal Docker network hostname `postgres`, not `localhost`), but it will confuse any host-side tooling. Either stop the local Postgres service, or map the compose Postgres to a different host port.

### Local development

```bash
task backend:dev   # regenerates swagger docs, then runs the app
```

Requires a reachable Postgres matching the `DB_*` / `GOOSE_*` vars in `.env`.

### Useful Taskfile commands

| Command | Purpose |
|---|---|
| `task backend:dev` | Run the app locally |
| `task db-migrate-up` / `db-migrate-down` | Apply/rollback migrations via the goose CLI |
| `task gen-model` | Regenerate `infra/db/model` from the live DB schema |
| `task gen-repo` | Regenerate each `core/*/repo.go` interface |
| `task gen-swagger` | Regenerate Swagger docs |
| `task backend:test:unit` | Run unit tests with coverage |

## Testing

```bash
task backend:test:unit
```

Runs `go test ./core/... -cover`. Most tests are pure unit tests using hand-written fakes for the `Repo`/`HISClient`/`TokenIssuer` interfaces. `core/*/repo_test.go` are integration tests against a real, migrated Postgres — they skip automatically (not fail) if no such database is reachable, and the DSN can be overridden via `TEST_DATABASE_DSN` or `GOOSE_DBSTRING`.

`.mockery.yaml` is present for generating interface mocks (`task backend:mock`), but depending on your `mockery`/Go toolchain versions it may hit an internal `go/packages` error — the hand-written fakes in the test files don't depend on it.
