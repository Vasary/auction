# Business Rules

This document is the source of truth for auction behavior. The backend enforces these rules even when clients race, reconnect, or submit malformed data. The frontend mirrors part of the validation only to improve the user experience.

## Auction Type

Auction Core implements a reverse auction.

| Rule | Description |
|---|---|
| Direction | Prices move downward |
| Winner | The company behind the last accepted, lowest bid |
| Price rule | A new bid must be strictly lower than the current price |
| Step rule | The decrease must match the configured auction step |
| Finality | Once an auction is finished, no further bids are accepted |

## Identifiers

All public IDs are UUIDs.

| Field | Meaning |
|---|---|
| `tenderId` | Auction identifier |
| `companyId` | Company participating in the auction |
| `personId` | Person acting on behalf of the company |

The HTTP layer rejects malformed UUIDs before they reach the domain code. WebSocket bid messages are also validated before the session attempts to process them.

## Money Representation

All prices, steps, and bids are represented as integer minor units.

Examples:

| Human value | Minor-unit value |
|---|---:|
| `100.00` | `10000` |
| `100.50` | `10050` |
| `1.00` | `100` |

The backend uses integer arithmetic for bid validation. Formatting into major units is a UI concern.

## Auction Statuses

| Status | Meaning | Bid Acceptance |
|---|---|---|
| `Scheduled` | Auction exists but has not started | Rejected |
| `Active` | Auction is open for bidding | Accepted if all rules pass |
| `Finished` | Auction is closed and final state is fixed | Rejected |

## Bid Validation Pipeline

The server processes every bid through a fixed validation sequence.

### 1. Message Shape

The incoming WebSocket message must contain:

- `type: "place_bid"`
- `bid`
- `companyId`
- `personId`

Missing identity fields are rejected before domain processing.

### 2. Participant Check

The company must be registered for the auction through:

```http
POST /auctions/{tenderId}/participate
```

Unregistered companies receive `not_a_participant`.

### 3. Active Auction Check

Only active auctions accept bids. Scheduled or finished auctions reject bids with an active-state error.

### 4. Per-Company Rate Limit

The session tracks the last successful bid time for each company. If a company submits another bid too quickly, the bid is rejected with `rate_limit_exceeded`.

### 5. Reverse-Auction Price Rule

The new bid must be strictly lower than the current price:

```text
newBid < currentPrice
```

If the bid is equal to or greater than the current price, it is rejected with `bid_must_be_lower`.

### 6. Step Alignment

The decrease must be large enough and aligned with the configured step:

```text
currentPrice - newBid >= step
(currentPrice - newBid) % step == 0
```

Violations are rejected with `bid_not_aligned_with_step`.

### 7. Persistence

After all domain checks pass, the bid is written to PostgreSQL. If persistence fails, the bid is rejected with `failed to persist bid`.

### 8. State Update and Broadcast

Only after persistence succeeds does the session:

1. Update `currentPrice`.
2. Update `winnerId`.
3. Store the latest bid in memory.
4. Broadcast `price_updated`.
5. Return an accepted `BidResult`.

## Concurrency Model

Each active auction is coordinated by one session. Bids for the same auction are processed through that session in deterministic order.

If two participants submit the same price at nearly the same time, one bid is processed first. The second bid is then compared against the updated current price and is rejected because it is no longer lower.

This approach keeps the business rules simple and protects the auction from race-condition winners.

## Error Reasons

| Reason | Meaning |
|---|---|
| `auction not active` | Auction is scheduled or otherwise not accepting bids |
| `auction finished` | Auction is already closed |
| `bid must be lower than current price` | Bid is not strictly lower than the current price |
| `bid not aligned with step` | Bid decrease does not satisfy the step rules |
| `rate limit exceeded` | Company submitted accepted bids too frequently |
| `not_a_participant` | Company is not registered for the auction |
| `missing company_id or person_id` | WebSocket message does not contain required identity fields |
| `bid must be at least 1` | Bid amount is non-positive |
| `failed to persist bid` | The bid could not be saved |

## Client Responsibilities

Clients should validate obvious mistakes before sending a bid, but the backend remains authoritative. A client may believe a bid is valid while another user has already lowered the price. In that case, the server rejection is correct and expected.

Recommended client behavior:

- Always render the latest `snapshot` or `price_updated` event.
- Treat rejected bids as normal business outcomes.
- Disable bid entry after `finished`.
- Reconnect and request fresh state after network interruptions.
