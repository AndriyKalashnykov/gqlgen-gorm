# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Project

Schema-first GraphQL **Todo** API in Go. [gqlgen](https://gqlgen.com/)
generates the server from `graph/typeDefs/todo.gql`; [GORM](https://gorm.io/)
persists to SQLite via the pure-Go `glebarez/sqlite` driver (no CGO). Ships as a
non-root `scratch` container.

## Layout

| Path | Purpose |
|------|---------|
| `server.go` | Entry point; wires the handler, `/healthz`, and the `-healthcheck` probe mode |
| `graph/typeDefs/todo.gql` | GraphQL schema (source of truth) |
| `graph/resolvers/` | Hand-written resolver implementations (`resolver.go`, `todo.resolvers.go`) |
| `graph/generated/`, `graph/customTypes/` | gqlgen output — **generated, gitignored**, never edit |
| `internal/common/` | DB init (`db.go`) and request-context plumbing (`context.go`) |
| `e2e/` | End-to-end tests (build tag `e2e`) |
| `gqlgen.yml` | gqlgen configuration |

## Toolchain

mise is the single source of truth (`.mise.toml`): Go **1.26**, golangci-lint,
trivy, gitleaks, hadolint, act, govulncheck. `make deps` installs everything.

The Go version is mirrored in `go.mod`, `.mise.toml`, and the Dockerfile
`ARG GO_VERSION`; `make check-go-alignment` (first dep of `static-check`) fails
if they drift, and Renovate groups the three so they bump together.

## Commands

| Command | Description |
|---------|-------------|
| `make deps` | Install the pinned toolchain |
| `make generate` | Regenerate gqlgen code (`go tool gqlgen generate`) |
| `make run` | Run the server on `:4000` |
| `make build` | Build the binary |
| `make test` | Unit tests (`-race`) |
| `make integration-test` | Integration tests (in-process gqlgen client + SQLite) |
| `make e2e` | E2E tests (real HTTP server, ephemeral port) |
| `make static-check` | Alignment + `go vet` + lint + govulncheck + Trivy + gitleaks + hadolint |
| `make ci` | Full local pipeline |
| `make image-build` / `image-run` | Build / run the container |

## Conventions

- gqlgen is registered as a Go `tool` directive; run it with `go tool`, never
  `go install ...@latest`. Generated code is gitignored and produced in CI and
  in the Docker build.
- After editing `graph/typeDefs/*.gql`, run `make generate`; resolver bodies in
  `graph/resolvers/*.go` are preserved across regeneration.
- Operator-tunable values are env-driven with `.env.example` defaults: `PORT`
  (server), `GQL_HOST`/`GQL_PORT` (`make todo-*` curls), `DB_DSN` (SQLite path).
- CI (`.github/workflows/ci.yml`): `changes → static-check → {build, test,
  integration-test, image-build}`; `e2e` needs `build` + `test`; `ci-pass`
  aggregates all seven jobs. `image-build` is build-only (validates the
  `scratch` image; no push). Actions are SHA-pinned; jobs use `jdx/mise-action`.
  `make ci` mirrors the CI job set (includes `image-build`).

## Gotchas

- The SQLite driver is **pure-Go** (`glebarez/sqlite`); do not switch back to
  `gorm.io/driver/sqlite` (mattn/go-sqlite3) — it needs CGO and breaks the
  `CGO_ENABLED=0` static `scratch` build.
- In the container the database lives at `/data/dev.db` (writable, owned by the
  non-root user); mount a volume at `/data` to persist it.

## Skills

These portfolio skills maintain this project's infrastructure files:

| Skill | Maintains |
|-------|-----------|
| `/makefile` | `Makefile`, `.mise.toml` |
| `/ci-workflow` | `.github/workflows/*.yml` |
| `/renovate` | `renovate.json` |
| `/readme` | `README.md` |
| `/project-review` | Runs all of the above in parallel |

When spawning subagents to review or modify these files, always pass the
relevant skill's full conventions into the agent prompt — agents cannot read
skill files themselves.
