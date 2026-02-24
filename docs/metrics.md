# Метрики и Observability

Документ описывает все метрики, которые приложение публикует на `GET /metrics` (Prometheus format), их назначение, labels и практическое применение.

## Ключевая идея

В проекте используется **Prometheus Histogram** (bucket-based) для latency/распределений вместо summary. Это означает:
- перцентили (`p50/p95/p99`) считаются в запросах PromQL через `histogram_quantile`;
- buckets агрегируются между инстансами, что удобно для горизонтального масштабирования;
- для каждой histogram Prometheus автоматически создает серии `_bucket`, `_sum`, `_count`.

## Группы метрик

### 1) Сессии аукционов и процесс

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `auction_active_total` | Gauge | - | Число активных аукционов в памяти | Контроль текущей нагрузки бизнес-процесса |
| `auction_managed_total` | Gauge | - | Всего загруженных в память сессий | Понимание размера runtime-состояния |
| `auction_connections_total` | Gauge | - | Общее число активных WS-подключений по всем сессиям | Быстрый индикатор real-time нагрузки |
| `auction_sessions_status_total` | GaugeVec | `status` (`Scheduled`, `Active`, `Finished`) | Распределение сессий по статусам | Видеть перекосы в lifecycle аукционов |
| `process_memory_alloc_bytes` | Gauge | - | Alloc из runtime.MemStats | Текущий рабочий heap |
| `process_memory_sys_bytes` | Gauge | - | Sys из runtime.MemStats | Общая память, взятая у ОС |

### 2) HTTP API

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `http_requests_total` | CounterVec | `method`, `route`, `status` | Количество HTTP-запросов | RPS, error-rate, разбивка по endpoint |
| `http_request_duration_seconds` | HistogramVec | `method`, `route`, `status` | Латентность HTTP-запросов | p95/p99, SLO по latency |
| `http_in_flight_requests` | Gauge | - | Текущее число запросов в обработке | Saturation и оценка очередей |

`route` берется из chi route pattern (например, `/auctions/{tenderId}`). Это снижает кардинальность по сравнению с raw URL.

### 3) WebSocket

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `ws_connections_active` | Gauge | - | Активные WS-соединения | Нагрузка на real-time слой |
| `ws_connect_total` | CounterVec | `result` (`ok`, `bad_request`, `not_found`, `upgrade_error`) | Исходы попыток подключения | Диагностика проблем handshаke/валидации |
| `ws_messages_total` | CounterVec | `direction` (`in`/`out`), `type`, `result` | Количество входящих/исходящих сообщений | Ошибки протокола и трафик по типам событий |
| `ws_message_duration_seconds` | HistogramVec | `type` | Время обработки входящего сообщения | p95/p99 по `place_bid` и др. |

### 4) Ставки и бизнес-исходы

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `auction_bids_total` | CounterVec | `result`, `reason` | Количество принятых/отклоненных ставок | Наблюдение за качеством процесса торгов |
| `auction_bid_amount` | Histogram | - | Распределение размера **принятых** ставок | Контроль бизнес-паттернов и аномалий |

Типовые значения `reason`:
- `none` (для accepted);
- `not_active`, `finished`, `rate_limited`, `not_lower`, `not_aligned`, `invalid_identity`, `not_participant`, `persist_error`, `unknown`, `other`.

### 5) Scheduler и recovery

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `scheduler_scans_total` | CounterVec | `result` (`ok`, `error`) | Результаты сканов планировщика | Мониторинг стабильности фоновой активации |
| `scheduler_found_auctions` | Histogram | - | Сколько аукционов найдено за scan | Профиль нагрузки планировщика |
| `scheduler_activation_total` | CounterVec | `result` (`activated`, `already_loaded`, `error`) | Итоги попыток активации | Детект silent-fail активаций |
| `auction_recovery_total` | CounterVec | `result` (`ok`, `error`) | Результат startup recovery | Быстрая проверка здоровья при старте |

### 6) База данных

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `db_query_total` | CounterVec | `op`, `result` (`ok`, `error`) | Количество DB-операций | Error-rate по операциям |
| `db_query_duration_seconds` | HistogramVec | `op` | Латентность DB-операций | p95/p99 для SQL-операций |
| `db_pool_acquired` | Gauge | - | Занятые коннекты пула | Saturation пула |
| `db_pool_idle` | Gauge | - | Idle коннекты | Capacity/резерв пула |
| `db_pool_total` | Gauge | - | Всего коннектов | Контроль лимитов пула |
| `db_pool_acquire_wait_seconds_total` | Counter | - | Накопленное время ожидания коннекта из пула | Идентификация нехватки DB connections |

Типичные `op`: `create_auction`, `get_auction_by_id`, `update_auction`, `update_auction_status`, `delete_auction`, `find_starting_between`, `list_auctions`, `create_bid_tx`, `get_bid_by_id`, `list_bids`, `add_participant`, `is_participant`, `list_participants`.

### 7) AMQP / RabbitMQ

| Metric | Type | Labels | Что показывает | Для чего использовать |
|---|---|---|---|---|
| `amqp_publish_total` | CounterVec | `routing_key`, `result` (`ok`, `error`) | Результаты публикаций в broker | Ошибки доставки событий |
| `amqp_publish_duration_seconds` | HistogramVec | `routing_key` | Латентность publish | p95/p99 для outbound event-потока |
| `amqp_connection_state` | Gauge | - | Состояние соединения (`1` up, `0` down) | Alert на потерю брокера |
| `amqp_reconnect_total` | CounterVec | `result` (`ok`, `error`) | Итоги reconnect попыток | Надежность восстановления связи |

## Перцентили и buckets

### Какие buckets используются
- Для latency-гистограмм (`http_request_duration_seconds`, `ws_message_duration_seconds`, `db_query_duration_seconds`, `amqp_publish_duration_seconds`) используется `prometheus.DefBuckets`.
- Для бизнес-распределений:
  - `auction_bid_amount`: `[1,10,50,100,500,1000,5000,10000,50000,100000,500000,1000000]`
  - `scheduler_found_auctions`: `[0,1,2,5,10,20,50,100]`

### Как считать percentile
Пример p95 HTTP latency за 5 минут:

```promql
histogram_quantile(
  0.95,
  sum by (le, route, method) (
    rate(http_request_duration_seconds_bucket[5m])
  )
)
```

Пример p99 DB latency за 10 минут:

```promql
histogram_quantile(
  0.99,
  sum by (le, op) (
    rate(db_query_duration_seconds_bucket[10m])
  )
)
```

## Полезные PromQL-запросы

### HTTP

RPS по route:

```promql
sum by (route, method) (rate(http_requests_total[1m]))
```

Error-rate (5xx):

```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

### WS

Текущие подключения:

```promql
ws_connections_active
```

Доля ошибок входящих сообщений:

```promql
sum(rate(ws_messages_total{direction="in",result="error"}[5m]))
/
sum(rate(ws_messages_total{direction="in"}[5m]))
```

### Ставки

Доля отклоненных ставок:

```promql
sum(rate(auction_bids_total{result="rejected"}[5m]))
/
sum(rate(auction_bids_total[5m]))
```

Rate-limit rejects:

```promql
sum(rate(auction_bids_total{result="rejected",reason="rate_limited"}[5m]))
```

### DB

p95 по операциям:

```promql
histogram_quantile(
  0.95,
  sum by (le, op) (rate(db_query_duration_seconds_bucket[5m]))
)
```

Доля DB ошибок:

```promql
sum(rate(db_query_total{result="error"}[5m]))
/
sum(rate(db_query_total[5m]))
```

Пул близок к исчерпанию:

```promql
db_pool_acquired / clamp_min(db_pool_total, 1)
```

### AMQP

Состояние соединения:

```promql
amqp_connection_state
```

Доля ошибок publish:

```promql
sum(rate(amqp_publish_total{result="error"}[5m]))
/
sum(rate(amqp_publish_total[5m]))
```

## Рекомендации по алертам

Минимальный набор:
- `http_5xx_rate > 2%` в течение 10 минут.
- `http p95 latency` выше вашего SLO порога (например, > 300ms) 10 минут.
- `db_query_total{result="error"}` заметно вырос относительно baseline.
- `db_pool_acquired / db_pool_total > 0.9` и рост `db_pool_acquire_wait_seconds_total`.
- `amqp_connection_state == 0` дольше 1-2 минут.
- аномальный рост `auction_bids_total{result="rejected"}` (особенно `persist_error` и `rate_limited`).

## Ограничения и важные замечания

- Метрики, которые зависят от in-memory состояния (`auction_*`, `process_memory_*`, DB pool gauges), обновляются во время scrape `/metrics`.
- Для горизонтально масштабируемого сервиса агрегируйте метрики по `instance`/`pod` через `sum`/`avg` на уровне PromQL.
- Из labels удалены высококардинальные идентификаторы (например `tender_id`) для безопасной эксплуатации в проде.
