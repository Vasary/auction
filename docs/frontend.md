# Frontend Developer Console

The frontend is a React + TypeScript developer console served by the Go backend. It is not a marketing website or a customer-facing auction portal; it is an inspection and testing surface for the auction engine.

## Technology Stack

| Area | Choice |
|---|---|
| Framework | React |
| Language | TypeScript |
| Build tool | Vite |
| Styling | Tailwind CSS |
| Components | Local primitives in `ui/src/components` |
| Runtime delivery | Built assets served by the Go HTTP server |

## Source Layout

```text
ui/src
  components          shared UI primitives and test panels
  lib                 small utility functions
  pages               top-level screens
  pages/auctions      auction list, login, and detail flows
```

Important files:

| File | Responsibility |
|---|---|
| `ui/src/pages/auctions/AuctionList.tsx` | Auction overview and CRUD entry point |
| `ui/src/pages/auctions/AuctionDetail.tsx` | Live auction screen with WebSocket behavior |
| `ui/src/pages/auctions/AuctionLogin.tsx` | Company/person identity selection |
| `ui/src/pages/E2ERunnerPage.tsx` | Functional test runner page |
| `ui/src/components/E2ERunnerPanel.tsx` | Multi-auction, multi-client scenario logic |

## WebSocket Lifecycle

The auction detail page connects to:

```text
ws://<host>/ws/{tenderId}
```

The client lifecycle is:

1. Open the WebSocket when the auction detail screen mounts.
2. Receive a `snapshot` and render current state.
3. Listen for `price_updated`, `bid_rejected`, `started`, and `finished`.
4. Send `place_bid` messages from the bid form.
5. Reconnect after interruptions and use server state as authoritative.

## Client-Side Validation

The UI performs lightweight validation before sending a bid:

- Required `companyId` and `personId`.
- Bid is below the locally known current price.
- Bid follows the known step.
- Bid input is disabled when the auction is finished.

This validation improves usability only. The backend remains the authority because another participant can change the price between local validation and message delivery.

## Identity Simulation

The console allows developers to change `companyId` and `personId` while testing. This makes it possible to simulate several participants by opening multiple tabs or browser windows.

This is especially useful for demonstrating:

- Participant registration requirements.
- Rate limiting per company.
- Price race behavior.
- WebSocket fan-out to all connected clients.

## Event Log

The auction detail screen displays incoming WebSocket events so developers can see the protocol in action.

Useful events during demos:

| Event | What to Look For |
|---|---|
| `snapshot` | Initial state after connection |
| `price_updated` | Accepted bid and new current price |
| `bid_rejected` | Participant registration failure |
| `finished` | Final winner and price |

## Bid History

The frontend reads persisted bid history through:

```http
GET /auctions/{tenderId}/bids
```

This lets the UI compare live events against durable state and helps reveal consistency issues during development.

## Building the UI

```bash
make build-ui
```

or directly:

```bash
cd ui
npm install
npm run build
```

The Go server serves the built UI under `/ui` and static assets under `/assets`.

## Role in the Project

The frontend makes the backend easy to evaluate. It turns the auction engine into a visible, interactive system where reviewers can see real-time coordination, error handling, participant simulation, and persisted state without writing custom scripts.
