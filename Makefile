SHELL := /bin/bash
.DEFAULT_GOAL := help

# Put mise shims (and the mise bin dir) on PATH so every tool pinned in
# .mise.toml resolves without a separate activation step.
export PATH := $(HOME)/.local/share/mise/shims:$(HOME)/.local/bin:$(PATH)
export GOFLAGS ?= -mod=mod
export GOTOOLCHAIN ?= local

# Operator-tunable values (mirror .env.example; override via env or .env).
GQL_HOST  ?= localhost
GQL_PORT  ?= 4000
GQL_URL   := http://$(GQL_HOST):$(GQL_PORT)/query

# Docker image coordinates.
DOCKER_IMAGE ?= gqlgen-gorm
DOCKER_TAG   ?= latest

# gqlgen is registered as a `tool` in go.mod, so `go tool` runs the exact
# pinned version and its codegen deps are recorded in go.sum.
GQLGEN := go tool github.com/99designs/gqlgen

.PHONY: help deps deps-docker check-go-alignment generate format vet lint vulncheck \
        trivy-fs secrets hadolint static-check build test integration-test e2e ci ci-run \
        clean run image-build image-run image-stop image-push renovate-validate \
        todo-create todo-update todo-get todo-get-all todo-delete

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

deps: ## Install the pinned toolchain via mise
	@command -v mise >/dev/null 2>&1 || { echo "mise not found — installing from https://mise.run"; curl -fsSL https://mise.run | sh; }
	@mise install --yes
	@mise reshim

deps-docker: ## Verify Docker is available (system prerequisite, not mise-managed)
	@command -v docker >/dev/null 2>&1 || { echo "docker not found — install Docker"; exit 1; }

check-go-alignment: ## Verify the Go version agrees across go.mod, .mise.toml and the Dockerfile
	@set -e; \
	gomod=$$(grep -oE '^go [0-9]+\.[0-9]+' go.mod | awk '{print $$2}'); \
	misev=$$(grep -oE '^go[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+' .mise.toml | grep -oE '[0-9]+\.[0-9]+'); \
	dockerv=$$(grep -oE '^ARG GO_VERSION=[0-9]+\.[0-9]+' Dockerfile | grep -oE '[0-9]+\.[0-9]+'); \
	if [ "$$gomod" != "$$misev" ] || [ "$$gomod" != "$$dockerv" ]; then \
		echo "ERROR: Go version disagrees across files:"; \
		printf "  %-12s %s\n" go.mod "$$gomod" .mise.toml "$$misev" Dockerfile "$$dockerv"; \
		exit 1; \
	fi; \
	echo "Go version aligned: $$gomod"

generate: deps ## Regenerate gqlgen code from the schema
	@rm -rf graph/customTypes graph/generated
	@$(GQLGEN) generate

format: deps ## Auto-format Go code
	@golangci-lint fmt ./...

vet: deps ## Run go vet
	@go vet ./...

lint: deps ## Run golangci-lint
	@golangci-lint run ./...
	@go mod tidy
	@git diff --exit-code go.mod go.sum

vulncheck: deps ## Scan for known Go vulnerabilities
	@govulncheck ./...

trivy-fs: deps ## Trivy filesystem scan (vuln, secret, misconfig)
	@trivy fs --scanners vuln,secret,misconfig --exit-code 1 --no-progress .

secrets: deps ## Scan the repo for committed secrets
	@gitleaks dir --no-banner --redact .

hadolint: deps ## Lint the Dockerfile
	@hadolint Dockerfile

static-check: check-go-alignment generate vet lint vulncheck trivy-fs secrets hadolint ## Run all static-analysis gates

build: generate ## Build the server binary
	@CGO_ENABLED=0 go build -o ./.bin/server server.go

test: generate ## Run unit tests with the race detector
	@go test -race -count=1 ./...

integration-test: generate ## Run integration tests (in-process gqlgen client + SQLite)
	@go test -race -count=1 -tags=integration ./...

e2e: generate ## Run end-to-end tests (real HTTP server over an ephemeral port)
	@go test -race -count=1 -tags=e2e ./e2e/...

ci: deps static-check test integration-test e2e build image-build ## Run the full local CI pipeline

ci-run: deps ## Run the GitHub Actions workflow locally via act
	@act push

run: generate ## Run the server locally
	@GQL_HOST=$(GQL_HOST) PORT=$(GQL_PORT) go run server.go

clean: ## Remove build artifacts and the dev database
	@rm -rf ./.bin/ graph/customTypes graph/generated dev.db

image-build: deps-docker generate ## Build the Docker image
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

image-run: deps-docker ## Run the Docker image
	@docker run --rm -p $(GQL_PORT):$(GQL_PORT) -e PORT=$(GQL_PORT) $(DOCKER_IMAGE):$(DOCKER_TAG)

image-stop: deps-docker ## Stop the running container
	@docker ps -q --filter ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG) | xargs -r docker stop

image-push: deps-docker ## Push the Docker image
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

renovate-validate: ## Validate renovate.json
	@npx --yes --package renovate -- renovate-config-validator renovate.json

todo-create: ## Create a todo via the running server
	@curl -s -H "Content-Type: application/json" -d '{ "query": "mutation {createTodo(text:\"todo\"){id text done}}" }' $(GQL_URL) | jq .

todo-update: ## Update a todo via the running server
	@curl -s -H "Content-Type: application/json" -d '{ "query": "mutation {updateTodo(input:{id:1,text:\"todo\",done:true}){id text done}}" }' $(GQL_URL) | jq .

todo-get-all: ## List todos via the running server
	@curl -s -H "Content-Type: application/json" -d '{ "query": "{getTodos{id text done}}" }' $(GQL_URL) | jq .

todo-get: ## Get a single todo via the running server
	@curl -s -H "Content-Type: application/json" -d '{ "query": "{getTodo(todoId:1){id text done}}" }' $(GQL_URL) | jq .

todo-delete: ## Delete a todo via the running server
	@curl -s -H "Content-Type: application/json" -d '{ "query": "mutation {deleteTodo(todoId:1){id text done}}" }' $(GQL_URL) | jq .
