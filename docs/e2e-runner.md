# E2E Runner

The E2E Runner is a browser-based functional test tool available in the developer UI at `/ui/e2e`. It exercises the running backend through HTTP and WebSocket APIs, then verifies that live events, final auction state, and persisted bid history agree.

## Purpose

The runner is built to validate behavior, not just availability.

It checks that:

- Auctions can be created through the API.
- Participants can be registered.
- Multiple WebSocket clients can connect to each auction.
- Bids are accepted or rejected according to domain rules.
- Final `winnerId` and `currentPrice` match the accepted bid stream.
- Persisted bid history matches the backend's final auction state.

## Source Files

| File | Responsibility |
|---|---|
| `ui/src/pages/E2ERunnerPage.tsx` | Page route and layout |
| `ui/src/components/E2ERunnerPanel.tsx` | Scenario orchestration, WebSocket clients, result validation |

## Multi-Client Simulation

For each auction, the runner creates `clientsPerAuction` virtual participants.

For every participant it:

1. Generates a `companyId` and `personId`.
2. Registers the company through `POST /auctions/{tenderId}/participate`.
3. Opens a dedicated WebSocket connection to `/ws/{tenderId}`.

Separate sockets are what make the scenario useful: the backend sees independent clients, not one script pretending after the fact.

## Time Handling

All timestamps are sent to the backend in UTC with `toISOString()`.

The scenario records UTC values for:

- `startAt`
- `endAt`

This avoids browser timezone ambiguity when comparing expected and actual results.

## Scenario Flow

For each generated auction, the runner:

1. Computes `startAt` and `endAt`.
2. Creates the auction with `POST /auctions`.
3. Registers all participants.
4. Connects all WebSocket clients.
5. Waits until the auction is active.
6. Sends bids in rounds using `bidRounds` and `bidIntervalMs`.
7. Tracks accepted `bid_result` events to compute expected final state.
8. Polls `GET /auctions/{tenderId}` until the auction is `Finished` or times out.
9. Loads persisted bids with `GET /auctions/{tenderId}/bids`.
10. Compares expected state, persisted state, and final auction state.

## Pass Criteria

An auction passes only when no validation reason is recorded.

The runner verifies:

- The auction reached `status=Finished`.
- At least one WebSocket client connected.
- Exactly `clientsPerAuction` clients connected.
- At least one bid attempt was made.
- At least one bid was accepted.
- At least one bid was persisted.
- Persisted bids strictly decrease by price.
- Differences between persisted bids follow the configured step.
- Expected winner and price exist when accepted bids exist.
- Expected winner matches the last persisted bid company.
- Expected price matches the last persisted bid amount.
- Final `winnerId` matches the last persisted bid company.
- Final `currentPrice` matches the last persisted bid amount.
- Final state matches the expected state computed from accepted bid results.

## Result Fields

Each auction result includes:

| Field | Meaning |
|---|---|
| `tenderId` | Auction ID |
| `startAt`, `endAt` | UTC schedule used by the test |
| `wsConnected` | Number of connected WebSocket clients |
| `expectedWinnerId` | Winner computed from accepted bid results |
| `expectedCurrentPrice` | Price computed from accepted bid results |
| `bidsAttempted` | Number of bid attempts sent |
| `bidsAccepted` | Number of accepted bid results observed |
| `bidsPersisted` | Number of bids returned by the backend history endpoint |
| `winnerId` | Final winner returned by the auction endpoint |
| `currentPrice` | Final price returned by the auction endpoint |
| `status` | Final auction status |
| `passed` | Whether all checks passed |
| `reasons[]` | Validation failures, if any |

## Why Strict Checks Matter

Soft E2E checks can produce false confidence. A scenario that creates auctions but connects zero WebSocket clients, persists zero bids, or never observes an accepted bid should not be considered successful.

The runner treats those cases as failures and explains why in `reasons[]`.

## Practical Settings

A stable local run usually starts with:

| Setting | Value |
|---|---|
| `auctionsCount` | `4` |
| `clientsPerAuction` | `3..4` |
| `startDelaySec` | `30..60` |
| `shortDurationSec` | `60` |
| `longDurationSec` | `120` |
| `bidRounds` | `2..3` |
| `bidIntervalMs` | `>= 550` when backend rate limit is near 500ms |

For stress exploration, change one dimension at a time. Increasing auctions, clients, bid rounds, and interval pressure together makes it harder to identify the actual bottleneck.

## Limitations

- This is a functional browser runner, not a replacement for k6, JMeter, or dedicated load testing.
- Large `auctionsCount * clientsPerAuction` values can hit browser or local machine limits.
- The scenario uses real time, so it is sensitive to backend, DB, RabbitMQ, and network delays.
- Very low `bidIntervalMs` values may intentionally trigger backend rate limits.
