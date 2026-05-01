# Auction Core

[![CI](https://github.com/Vasary/auction/actions/workflows/container.yml/badge.svg?branch=main)](https://github.com/Vasary/auction/actions/workflows/container.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Vasary/auction)](https://goreportcard.com/report/github.com/Vasary/auction)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Vasary/auction?filename=go.mod)](go.mod)
[![License](https://img.shields.io/badge/license-attribution_required-blue)](LICENSE)

Auction Core is a production-minded Go service for real-time reverse auctions. It combines a clean domain core, WebSocket bidding, PostgreSQL persistence, RabbitMQ domain events, Prometheus metrics, and a React developer console for manual and functional testing.

The project is designed as a portfolio-grade example of backend engineering in Go: explicit business rules, deterministic validation, observable runtime behavior, focused tests, and a Docker-ready delivery pipeline.

## Highlights

- Real-time reverse auction engine with in-memory sessions and persisted bid history.
- Strong domain boundaries around auction lifecycle, bidding, participants, storage, transport, and scheduling.
- WebSocket protocol for live snapshots, bid results, price updates, and final state broadcasts.
- PostgreSQL repositories built on `pgx`.
- RabbitMQ publishing for auction lifecycle events.
- Prometheus metrics for HTTP, WebSocket, bid outcomes, DB operations, scheduler activity, AMQP, and runtime state.
- React + TypeScript developer UI served by the Go application.
- Browser-based E2E runner that simulates multiple auctions and multiple WebSocket clients.
- GitHub Actions CI that runs Go tests before building and publishing a container image.

## Architecture

```text
cmd/server                              application entrypoint and dependency wiring
internal/auction                        domain model, manager, in-memory sessions, bidding rules
internal/http                           REST API, WebSocket endpoint, static UI serving
internal/scheduler                      scheduled auction activation
internal/db                             PostgreSQL pool setup
internal/auction/repository/postgres    pgx-backed persistence
internal/amqp                           RabbitMQ connection and event publishing
internal/metrics                        Prometheus instrumentation
ui                                      React + TypeScript developer console
docs                                    architecture, business rules, metrics, and testing docs
```

## Documentation

- [System Flow and Developer UI](docs/flow.md)
- [Business Rules](docs/business-rules.md)
- [Backend Architecture](docs/backend.md)
- [Frontend Developer Console](docs/frontend.md)
- [Metrics and Observability](docs/metrics.md)
- [E2E Runner](docs/e2e-runner.md)

## Core Concepts

Auction Core implements reverse auctions: every accepted bid must lower the current price by an allowed step. All money values are stored as integer minor units, so the backend never depends on floating-point arithmetic for price decisions.

All public identifiers use UUIDs:

- `tenderId` identifies an auction.
- `companyId` identifies a participating company.
- `personId` identifies the person acting for that company.

## HTTP and WebSocket Surface

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/auctions` | Create an auction |
| `GET` | `/auctions` | List auctions |
| `GET` | `/auctions/{tenderId}` | Get one auction |
| `PATCH` | `/auctions/{tenderId}` | Update a scheduled auction |
| `DELETE` | `/auctions/{tenderId}` | Delete a scheduled auction |
| `POST` | `/auctions/{tenderId}/participate` | Register a company as a participant |
| `GET` | `/auctions/{tenderId}/bids` | Read bid history |
| `GET` | `/ws/{tenderId}` | Join the real-time auction stream |
| `GET` | `/ui` | Developer console |

## Quick Start

### Requirements

- Go 1.25+
- PostgreSQL
- RabbitMQ
- Node.js 18+ when rebuilding the UI locally

### Run Tests

```bash
go test ./...
```

### Run Locally

Set the required infrastructure URLs:

```bash
export DATABASE_URL='postgres://auction:password@localhost:5432/auction_db?sslmode=disable'
export AMQP_URL='amqp://rabbit:password@localhost:5672/'
export PORT=8082
```

Start the server:

```bash
go run ./cmd/server
```

The developer UI is available at:

```text
http://localhost:8082/ui
```

### Makefile

The repository includes a small Makefile for common workflows:

```bash
make help          # show available targets
make test          # run Go tests
make build-ui      # install UI dependencies and build the React app
make run-server    # run the Go server
make run           # build UI, then run the server
make docker-build  # build the Docker image
```

## CI and Delivery

The `Container` GitHub Actions workflow runs on pushes and pull requests for `main` and `master`. It executes `go test ./...` first; the container build only runs after the test job succeeds. On non-PR events, the Docker image is published to GitHub Container Registry with branch, tag, semantic version, SHA, and `latest` tags where applicable.

## License

This project is available under an attribution-required license. You may use, copy, modify, and distribute the code, but public use or redistribution must include visible credit and a link to [Viktor Gievoi](https://github.com/Vasary). See [LICENSE](LICENSE) for details.

## Why This Project Matters

This codebase demonstrates practical backend engineering beyond CRUD:

- Stateful real-time coordination with deterministic ordering per auction session.
- Domain-level validation that protects the system even when clients race or reconnect.
- Persistence and live broadcasts kept consistent through one bidding path.
- Operational visibility built into the service rather than bolted on later.
- A developer console and E2E runner that make the system easy to inspect and demo.
