# syntax=docker/dockerfile:1
# Keep GO_VERSION in sync with go.mod (`go` directive) and .mise.toml.
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.26
ARG APP_INTERNAL_PORT=4000
ARG APP_UID=10001
ARG APP_GID=10001

# Build the server binary (gqlgen code is generated inside the image so the
# build does not depend on committed generated sources).
FROM golang:${GO_VERSION} AS builder
WORKDIR /source

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Generate gqlgen code, build the static binary, and pre-create a writable data
# dir owned by the non-root runtime user (so the scratch image can create its
# SQLite database). One layer to keep the build hermetic and hadolint-clean.
RUN go tool github.com/99designs/gqlgen generate \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server server.go \
    && mkdir -p /data

FROM scratch
ARG APP_INTERNAL_PORT
ARG APP_UID
ARG APP_GID

ENV PORT=${APP_INTERNAL_PORT}
ENV DB_DSN=/data/dev.db

COPY --from=builder /server /server
COPY --from=builder --chown=${APP_UID}:${APP_GID} /data /data

EXPOSE ${APP_INTERNAL_PORT}
USER ${APP_UID}:${APP_GID}
WORKDIR /data

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "-healthcheck"]

ENTRYPOINT ["/server"]
