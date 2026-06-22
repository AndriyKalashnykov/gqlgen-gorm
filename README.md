[![CI](https://github.com/AndriyKalashnykov/gqlgen-gorm/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/AndriyKalashnykov/gqlgen-gorm/actions/workflows/ci.yml)
[![Hits](https://hits.sh/github.com/AndriyKalashnykov/gqlgen-gorm.svg?view=today-total&style=plastic)](https://hits.sh/github.com/AndriyKalashnykov/gqlgen-gorm/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Renovate enabled](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://app.renovatebot.com/dashboard#github/AndriyKalashnykov/gqlgen-gorm)

# Golang + GraphQL + GORM — Schema-First Todo API

A schema-first GraphQL **Todo** API written in Go. The **runtime surface** is a
[gqlgen](https://gqlgen.com/)-generated server — an interactive GraphQL
Playground, a single `POST /query` endpoint, and a `/healthz` probe — with
[GORM](https://gorm.io/) persisting to SQLite over the pure-Go
[glebarez/sqlite](https://github.com/glebarez/sqlite) driver (no CGO). The
**delivery surface** is a tiny (~22 MB) non-root `scratch` image with a
HEALTHCHECK, a [mise](https://mise.jdx.dev/)-pinned toolchain, a three-layer
test pyramid (unit / integration / e2e), and a CVE-gated GitHub Actions pipeline
(govulncheck, Trivy, gitleaks, hadolint). The schema lives in
[`graph/typeDefs/todo.gql`](graph/typeDefs/todo.gql).

## Table of Contents

- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [API](#api)
  - [createTodo](#createtodo)
  - [getTodo](#gettodo)
  - [getTodos](#gettodos)
  - [updateTodo](#updatetodo)
  - [deleteTodo](#deletetodo)
- [Docker](#docker)
- [Make Targets](#make-targets)
- [CI/CD](#cicd)
- [License](#license)

## Tech Stack

| Component   | Technology                                                              |
|-------------|-------------------------------------------------------------------------|
| Language    | Go 1.26                                                                  |
| GraphQL     | [gqlgen](https://github.com/99designs/gqlgen) (schema-first code generation) |
| GraphQL AST | [gqlparser/v2](https://github.com/vektah/gqlparser)                      |
| ORM         | [GORM](https://gorm.io/)                                                 |
| Database    | SQLite via the pure-Go [glebarez/sqlite](https://github.com/glebarez/sqlite) driver (no CGO) |
| Container   | Multi-stage Docker → `scratch` (non-root, HEALTHCHECK)                   |
| Toolchain   | [mise](https://mise.jdx.dev/)                                            |
| Build       | GNU Make                                                                 |

## Quick Start

```bash
make deps     # install the pinned toolchain via mise
make run      # generate code and start the server on :4000
```

Then open the GraphQL Playground:

```bash
xdg-open http://localhost:4000/
```

The bind port is configurable via `PORT` (or `GQL_PORT` for the `make` targets);
the SQLite path via `DB_DSN`. Defaults live in [`.env.example`](.env.example).

## Prerequisites

| Tool   | Version | Purpose                                                  |
|--------|---------|----------------------------------------------------------|
| [mise](https://mise.jdx.dev/) | latest  | Provisions Go and the dev tools (`make deps`) |
| Go     | 1.26    | Build/run the server (installed by mise)                 |
| GNU Make | any   | Task runner                                              |
| `jq`   | any     | Pretty-prints the `make todo-*` curl responses           |
| Docker | any     | Build/run the container image (optional)                 |

All tool versions are pinned in [`.mise.toml`](.mise.toml); `make deps`
installs them.

## API

All operations are served over `POST /query`. The `make todo-*` targets below
issue the equivalent `curl` calls (requires the server to be running and `jq`).

### createTodo

```bash
make todo-create
```

```graphql
mutation createTodo {
    createTodo(text: "todo1") {
        id
        text
        done
    }
}
```

```json
{ "data": { "createTodo": { "id": 1, "text": "todo1", "done": false } } }
```

### getTodo

```bash
make todo-get
```

```graphql
query getTodo {
    getTodo(todoId: 1) {
        id
        text
        done
    }
}
```

```json
{ "data": { "getTodo": { "id": 1, "text": "todo1", "done": false } } }
```

### getTodos

```bash
make todo-get-all
```

```graphql
query getTodos {
    getTodos {
        id
        text
        done
    }
}
```

```json
{ "data": { "getTodos": [ { "id": 1, "text": "todo1", "done": false } ] } }
```

### updateTodo

```bash
make todo-update
```

```graphql
mutation updateTodo {
    updateTodo(input: { id: 1, text: "todo", done: true }) {
        id
        text
        done
    }
}
```

```json
{ "data": { "updateTodo": { "id": 1, "text": "todo", "done": true } } }
```

### deleteTodo

Deletes the row and returns the record as it was before deletion.

```bash
make todo-delete
```

```graphql
mutation deleteTodo {
    deleteTodo(todoId: 1) {
        id
        text
        done
    }
}
```

```json
{ "data": { "deleteTodo": { "id": 1, "text": "todo", "done": true } } }
```

## Docker

```bash
make image-build      # build the scratch image
make image-run        # run it on :4000 (HEALTHCHECK probes /healthz)
```

The image runs as a non-root user and stores the SQLite database under
`/data` (mount a volume there to persist data across restarts).

## Make Targets

Run `make help` for the full list. Common targets:

| Target             | Description                                            |
|--------------------|--------------------------------------------------------|
| `deps`             | Install the pinned toolchain via mise                  |
| `generate`         | Regenerate gqlgen code from the schema                 |
| `run`              | Run the server locally                                 |
| `build`            | Build the server binary                                |
| `test`             | Unit tests (`-race`)                                   |
| `integration-test` | Integration tests (in-process gqlgen client + SQLite)  |
| `e2e`              | End-to-end tests (real HTTP server, ephemeral port)    |
| `static-check`     | Alignment + lint + `go vet` + govulncheck + Trivy + gitleaks + hadolint |
| `ci`               | Full local pipeline                                    |

## CI/CD

GitHub Actions runs on every push to `main`, tags `v*`, and pull requests
([`.github/workflows/ci.yml`](.github/workflows/ci.yml)). A `changes` detector
gates the heavy jobs on code changes; `ci-pass` aggregates them into one
required status. Tool versions come from `.mise.toml` via `jdx/mise-action`;
all actions are SHA-pinned.

| Job | Needs | Runs |
|-----|-------|------|
| `changes` | — | `dorny/paths-filter` — sets `code` output |
| `static-check` | `changes` | `make static-check` (alignment, `go vet`, golangci-lint, govulncheck, Trivy, gitleaks, hadolint) |
| `build` | `static-check` | `make build` |
| `test` | `static-check` | `make test` (unit, `-race`) |
| `integration-test` | `static-check` | `make integration-test` (in-process gqlgen client + SQLite) |
| `e2e` | `build`, `test` | `make e2e` (real HTTP server, ephemeral port) |
| `image-build` | `static-check` | `make image-build` (build-only validation of the `scratch` image) |
| `ci-pass` | all of the above | Aggregator — the single required check |

No repository secrets are required (the workflow uses the built-in
`GITHUB_TOKEN` only). [Renovate](https://docs.renovatebot.com/) keeps
dependencies current, with the Go version grouped across `go.mod`,
`.mise.toml`, and the Dockerfile so it bumps in lockstep. A weekly
[`cleanup-runs.yml`](.github/workflows/cleanup-runs.yml) prunes old workflow runs
(per-workflow retention) and stale branch caches.

## License

[MIT](LICENSE)
