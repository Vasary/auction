# System Flow and Developer UI

Auction Core is built around a simple flow: create an auction, register participants, activate the session, process live bids, and close the auction with a durable final result.

## End-to-End Flow

### 1. Create an Auction

An administrator, integration test, or external service creates an auction through the HTTP API. The request defines:

- `tenderId`
- `startPrice`
- `step`
- `startAt`
- `endAt`
- `createdBy`

The auction starts as `Scheduled`.

### 2. Register Participants

Companies must opt into a specific auction before bidding:

```http
POST /auctions/{tenderId}/participate
```

The WebSocket layer checks participant status before allowing a bid into the session.

### 3. Activate the Auction

An auction can become active in two ways:

- The scheduler detects that `startAt` has arrived.
- The auction status is updated explicitly through the API.

When activation succeeds, the manager creates or loads an in-memory session and publishes a lifecycle event through RabbitMQ.

### 4. Join Through WebSocket

Participants connect to:

```http
GET /ws/{tenderId}
```

Each client receives an immediate `snapshot`, then live events such as `price_updated`, `bid_rejected`, and `finished`.

### 5. Place Bids

Clients send `place_bid` messages containing a bid amount and identity fields. The backend validates participation, status, rate limits, price direction, step alignment, and persistence.

Accepted bids update the current price and winner. Rejected bids return an explicit reason.

### 6. Finish the Auction

When `endAt` arrives or the auction is closed explicitly, the final winner and price are fixed. The session stops accepting bids and broadcasts `finished`.

## Developer UI

The built-in UI at `/ui` is a developer console. It is intentionally practical: it exists to inspect the system, test WebSocket behavior, and demonstrate business rules without needing separate tooling.

## Main Screens

### Auction List

The list screen provides a quick overview of auctions stored in PostgreSQL.

Typical actions:

- Create a new auction.
- Filter or inspect auctions by status.
- Open a detail screen.
- Update or delete scheduled test data.

### Auction Detail

The detail screen is the live bidding workbench.

It shows:

- Current auction status.
- Current price.
- Winner company ID.
- WebSocket connection state.
- Incoming events.
- Persisted bid history.

It also provides a bid form that can act as different companies and people, which makes it easy to simulate multiple participants across browser tabs.

### E2E Runner

The `/ui/e2e` screen runs a functional scenario against the running backend. It creates auctions, registers participants, opens several WebSocket clients, sends bids, waits for final state, and verifies consistency between WebSocket events, HTTP reads, and persisted bid history.

See [E2E Runner](e2e-runner.md) for the full validation model.

## Demo Script

For a concise demo:

1. Start the backend with PostgreSQL and RabbitMQ configured.
2. Open `/ui`.
3. Create a scheduled auction with a short duration.
4. Register two or more companies.
5. Open the auction in multiple tabs with different `companyId` values.
6. Submit valid and invalid bids to show rule enforcement.
7. Watch the event log update in real time.
8. Let the auction finish and confirm the final winner and price.

## Why the Flow Is Useful

This flow demonstrates the parts that matter in a real system:

- Business rules are enforced server-side.
- WebSocket clients receive live state changes.
- Bid history remains durable.
- Concurrent users cannot bypass price ordering.
- Operational signals are available through `/metrics`.
