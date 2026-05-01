# Metrics and Observability

Auction Core exposes Prometheus metrics at:

```http
GET /metrics
```

The metrics cover transport behavior, auction outcomes, database operations, scheduler work, RabbitMQ publishing, and runtime state. They are designed to answer practical production questions: Is the API healthy? Are bids being rejected? Is the DB pool saturated? Are lifecycle events publishing? Are WebSocket clients connected?

## Histogram Strategy

Latency and distribution metrics use Prometheus histograms.

This means:

- Percentiles are calculated in PromQL with `histogram_quantile`.
- Buckets can be aggregated across service instances.
- Prometheus automatically creates `_bucket`, `_sum`, and `_count` series.

## Auction Runtime Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `auction_active_total` | Gauge | none | Active auctions currently held in memory |
| `auction_managed_total` | Gauge | none | Total sessions managed in memory |
| `auction_connections_total` | Gauge | none | Active WebSocket connections across sessions |
| `auction_sessions_status_total` | GaugeVec | `status` | Sessions grouped by `Scheduled`, `Active`, `Finished` |
| `process_memory_alloc_bytes` | Gauge | none | Current allocated heap from `runtime.MemStats` |
| `process_memory_sys_bytes` | Gauge | none | Total memory obtained from the OS |

## HTTP Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `http_requests_total` | CounterVec | `method`, `route`, `status` | Request count by endpoint and status |
| `http_request_duration_seconds` | HistogramVec | `method`, `route`, `status` | Request latency |
| `http_in_flight_requests` | Gauge | none | Requests currently being handled |

The `route` label uses the chi route pattern, such as `/auctions/{tenderId}`, to avoid high-cardinality raw URLs.

## WebSocket Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `ws_connections_active` | Gauge | none | Active WebSocket connections |
| `ws_connect_total` | CounterVec | `result` | Connection attempts grouped by outcome |
| `ws_messages_total` | CounterVec | `direction`, `type`, `result` | Incoming and outgoing WebSocket message counts |
| `ws_message_duration_seconds` | HistogramVec | `type` | Processing latency for incoming messages |

Typical `ws_connect_total` results include `ok`, `bad_request`, `not_found`, and `upgrade_error`.

## Bid Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `auction_bids_total` | CounterVec | `result`, `reason` | Accepted and rejected bid outcomes |
| `auction_bid_amount` | Histogram | none | Distribution of accepted bid amounts |

Typical rejection reasons:

- `not_active`
- `finished`
- `rate_limited`
- `not_lower`
- `not_aligned`
- `invalid_identity`
- `not_participant`
- `persist_error`
- `unknown`
- `other`

## Scheduler and Recovery Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `scheduler_scans_total` | CounterVec | `result` | Scheduler scan outcomes |
| `scheduler_found_auctions` | Histogram | none | Number of auctions found per scan |
| `scheduler_activation_total` | CounterVec | `result` | Activation attempts grouped by outcome |
| `auction_recovery_total` | CounterVec | `result` | Startup recovery result |

## Database Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `db_query_total` | CounterVec | `op`, `result` | DB operation count |
| `db_query_duration_seconds` | HistogramVec | `op` | DB operation latency |
| `db_pool_acquired` | Gauge | none | Connections currently acquired |
| `db_pool_idle` | Gauge | none | Idle connections |
| `db_pool_total` | Gauge | none | Total pool connections |
| `db_pool_acquire_wait_seconds_total` | Counter | none | Cumulative time spent waiting for a connection |

Common `op` values include `create_auction`, `get_auction_by_id`, `update_auction`, `update_auction_status`, `delete_auction`, `find_starting_between`, `list_auctions`, `create_bid_tx`, `get_bid_by_id`, `list_bids`, `add_participant`, `is_participant`, and `list_participants`.

## AMQP Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `amqp_publish_total` | CounterVec | `routing_key`, `result` | RabbitMQ publish outcomes |
| `amqp_publish_duration_seconds` | HistogramVec | `routing_key` | Publish latency |
| `amqp_connection_state` | Gauge | none | `1` when connected, `0` when disconnected |
| `amqp_reconnect_total` | CounterVec | `result` | Reconnect attempts |

## Buckets

Latency histograms use `prometheus.DefBuckets`.

Business histograms use domain-specific buckets:

```text
auction_bid_amount:
[1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000]

scheduler_found_auctions:
[0, 1, 2, 5, 10, 20, 50, 100]
```

## PromQL Examples

### HTTP RPS by Route

```promql
sum by (route, method) (rate(http_requests_total[1m]))
```

### HTTP 5xx Error Rate

```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

### HTTP p95 Latency

```promql
histogram_quantile(
  0.95,
  sum by (le, route, method) (
    rate(http_request_duration_seconds_bucket[5m])
  )
)
```

### Active WebSocket Connections

```promql
ws_connections_active
```

### WebSocket Incoming Error Ratio

```promql
sum(rate(ws_messages_total{direction="in",result="error"}[5m]))
/
sum(rate(ws_messages_total{direction="in"}[5m]))
```

### Rejected Bid Ratio

```promql
sum(rate(auction_bids_total{result="rejected"}[5m]))
/
sum(rate(auction_bids_total[5m]))
```

### Rate-Limit Rejections

```promql
sum(rate(auction_bids_total{result="rejected",reason="rate_limited"}[5m]))
```

### DB p95 by Operation

```promql
histogram_quantile(
  0.95,
  sum by (le, op) (rate(db_query_duration_seconds_bucket[5m]))
)
```

### DB Error Ratio

```promql
sum(rate(db_query_total{result="error"}[5m]))
/
sum(rate(db_query_total[5m]))
```

### DB Pool Saturation

```promql
db_pool_acquired / clamp_min(db_pool_total, 1)
```

### RabbitMQ Publish Error Ratio

```promql
sum(rate(amqp_publish_total{result="error"}[5m]))
/
sum(rate(amqp_publish_total[5m]))
```

## Alert Suggestions

| Alert | Suggested Signal |
|---|---|
| HTTP error rate | 5xx rate above 2% for 10 minutes |
| HTTP latency | p95 above the service SLO for 10 minutes |
| DB errors | `db_query_total{result="error"}` rising above baseline |
| DB saturation | `db_pool_acquired / db_pool_total > 0.9` with growing acquire wait |
| RabbitMQ disconnected | `amqp_connection_state == 0` for more than 1-2 minutes |
| Bid rejection spike | Sharp increase in rejected bids, especially `persist_error` |

## Operational Notes

- In-memory metrics are updated when `/metrics` is scraped.
- High-cardinality identifiers such as `tenderId` are intentionally excluded from labels.
- In multi-instance deployments, aggregate by `instance` or `pod` in PromQL depending on the question.
