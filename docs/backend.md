# Backend Architecture

Auction Core is a Go service that keeps the business-critical auction loop small, explicit, and testable. The backend exposes an HTTP API, a WebSocket bidding protocol, Prometheus metrics, and a static developer console built from the `ui` package.

## Design Goals

- Keep auction rules inside the domain layer, away from transport and storage details.
- Process bids deterministically for each auction session.
- Persist the source of truth in PostgreSQL while keeping active auction state fast in memory.
- Publish lifecycle events to RabbitMQ for downstream systems.
- Make production behavior visible through metrics, structured logs, and health checks.

## Runtime Components

| Component | Path | Responsibility |
|---|---|---|
| Server entrypoint | `cmd/server/main.go` | Configuration, dependency wiring, recovery, scheduler startup, graceful shutdown |
| HTTP transport | `internal/http` | REST routes, JSON responses, UUID validation, UI/static serving |
| WebSocket transport | `internal/http/ws.go`, `internal/ws` | Client lifecycle, bid messages, event delivery |
| Domain core | `internal/auction` | Auction model, manager, session loop, bidding rules |
| Scheduler | `internal/scheduler` | Periodically activates auctions whose `startAt` has arrived |
| PostgreSQL | `internal/db`, `internal/auction/repository/postgres` | Connection pool and pgx repositories |
| AMQP | `internal/amqp` | RabbitMQ connection management and lifecycle event publishing |
| Metrics | `internal/metrics` | Prometheus collectors and HTTP middleware |

## Application Startup

1. The server initializes the zap logger and timezone behavior.
2. `DATABASE_URL` is required and used to create the PostgreSQL pool.
3. `AMQP_URL` is used to connect the RabbitMQ publisher.
4. Repositories are created for auctions, bids, and participants.
5. `auction.Manager` is initialized and recovers active or recently relevant sessions.
6. The scheduler starts in the background and scans for auctions to activate.
7. The HTTP server starts on `PORT`, defaulting to `8082`.
8. On interrupt, the service performs graceful HTTP shutdown.

## Domain Model

The main domain objects live in `internal/auction/models.go`.

| Type | Purpose |
|---|---|
| `PersistedAuction` | Stored auction record loaded from PostgreSQL |
| `Snapshot` | Public runtime state sent to WebSocket clients |
| `Bid` | Persisted bid history item |
| `LatestBid` | Last accepted bid embedded into snapshots |
| `Status` | `Scheduled`, `Active`, or `Finished` |
| `Event` | Server-to-client WebSocket event envelope |
| `BidResult` | Client-facing response for bid attempts |

## Session Lifecycle

An active auction is represented by an in-memory `Session`.

1. A scheduled auction is created through `POST /auctions`.
2. Participants are registered through `/auctions/{tenderId}/participate`.
3. The scheduler or an explicit status update activates the auction.
4. The manager creates or loads a session.
5. WebSocket clients join the session and receive an immediate `snapshot`.
6. Bid messages enter the session, where validation and state updates happen in one ordered path.
7. Accepted bids are persisted, reflected in memory, and broadcast to connected clients.
8. At completion, the session publishes `finished`, stores final state, and stops accepting bids.

## WebSocket Protocol

### Connect

```http
GET /ws/{tenderId}
```

The server validates the auction ID, upgrades the connection, registers the client, and sends the current `snapshot`.

### Client Message

```json
{
  "type": "place_bid",
  "bid": 95000,
  "companyId": "550e8400-e29b-41d4-a716-446655440001",
  "personId": "550e8400-e29b-41d4-a716-446655440002"
}
```

### Server Events

| Event | Meaning |
|---|---|
| `snapshot` | Full current auction state |
| `started` | Auction became active |
| `price_updated` | A bid was accepted and the current price changed |
| `bid_rejected` | The company is not registered as an auction participant |
| `finished` | Auction ended and final state is fixed |

### Bid Result

```json
{
  "accepted": true,
  "currentPrice": 95000,
  "winnerId": "550e8400-e29b-41d4-a716-446655440001"
}
```

Rejected domain bids return `accepted: false` with a stable error string. Participant registration failures are sent as a `bid_rejected` event.

## HTTP API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Service health check |
| `GET` | `/metrics` | Prometheus exposition endpoint |
| `POST` | `/auctions` | Create a new scheduled auction |
| `GET` | `/auctions` | List all auctions |
| `GET` | `/auctions/{tenderId}` | Read one auction |
| `PATCH` | `/auctions/{tenderId}` | Update a scheduled auction |
| `DELETE` | `/auctions/{tenderId}` | Delete a scheduled auction |
| `POST` | `/auctions/{tenderId}/participate` | Register a company for an auction |
| `GET` | `/auctions/{tenderId}/bids` | List persisted bids |
| `GET` | `/ws/{tenderId}` | Open the auction WebSocket stream |
| `GET` | `/ui` | Serve the developer console |

## Data and Consistency

Auction state is intentionally split:

- PostgreSQL is the durable source of truth for auctions, participants, and bid history.
- Active sessions keep the current bidding state in memory for fast validation and fan-out.
- The manager is responsible for creating, recovering, and stopping sessions.
- Accepted bids follow one path: validate, persist, update memory, broadcast.

That ordering keeps WebSocket clients aligned with persisted state and makes failed persistence visible instead of silently accepting a bid that cannot be stored.

## Observability

The backend exposes metrics for HTTP requests, WebSocket traffic, bid outcomes, DB operations, scheduler scans, recovery, AMQP publishing, and runtime memory. See [Metrics and Observability](metrics.md) for PromQL examples and alert suggestions.

## Testing Focus

The backend has unit and integration-style tests around:

- HTTP handlers and UUID validation.
- Auction creation, update, deletion, listing, and bid history.
- Participation rules.
- Auction manager and session behavior.
- Scheduler activation behavior.
- WebSocket client mechanics.
- AMQP publisher behavior.

Run the suite with:

```bash
go test ./...
```
