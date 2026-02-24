package metrics

import (
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"auction-core/internal/auction"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ActiveAuctions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_active_total",
		Help: "Total number of currently active auctions.",
	})

	ManagedAuctions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_managed_total",
		Help: "Total number of auctions currently loaded in memory.",
	})

	AuctionConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_connections_total",
		Help: "Total number of active websocket connections.",
	})

	AuctionsByStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "auction_sessions_status_total",
		Help: "Number of in-memory auction sessions by status.",
	}, []string{"status"})

	MemoryAlloc = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_memory_alloc_bytes",
		Help: "Number of bytes allocated and still in use.",
	})

	MemorySys = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_memory_sys_bytes",
		Help: "Number of bytes obtained from system.",
	})

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests.",
	}, []string{"method", "route", "status"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})

	HTTPInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "Current number of in-flight HTTP requests.",
	})

	WSConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_active",
		Help: "Current number of active websocket connections.",
	})

	WSConnectTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_connect_total",
		Help: "Total websocket connection attempts grouped by result.",
	}, []string{"result"})

	WSMessagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_total",
		Help: "Total websocket messages grouped by direction, type and result.",
	}, []string{"direction", "type", "result"})

	WSMessageDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ws_message_duration_seconds",
		Help:    "Websocket message handling duration in seconds by message type.",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	AuctionBidsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_bids_total",
		Help: "Total bids grouped by acceptance and rejection reason.",
	}, []string{"result", "reason"})

	AuctionBidAmount = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "auction_bid_amount",
		Help:    "Distribution of accepted bid amounts.",
		Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
	})

	SchedulerScansTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "scheduler_scans_total",
		Help: "Total scheduler scans grouped by result.",
	}, []string{"result"})

	SchedulerFoundAuctions = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "scheduler_found_auctions",
		Help:    "Number of auctions found per scheduler scan.",
		Buckets: []float64{0, 1, 2, 5, 10, 20, 50, 100},
	})

	SchedulerActivationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "scheduler_activation_total",
		Help: "Total scheduler activation attempts grouped by result.",
	}, []string{"result"})

	RecoveryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_recovery_total",
		Help: "Auction recovery outcomes grouped by result.",
	}, []string{"result"})

	DBQueryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "db_query_total",
		Help: "Total database operations grouped by operation and result.",
	}, []string{"op", "result"})

	DBQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Database operation latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"op"})

	DBPoolAcquired = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_acquired",
		Help: "Database pool acquired connections.",
	})

	DBPoolIdle = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_idle",
		Help: "Database pool idle connections.",
	})

	DBPoolTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_total",
		Help: "Database pool total connections.",
	})

	DBPoolAcquireWaitSeconds = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_acquire_wait_seconds_total",
		Help: "Total time spent waiting for database connections from pool.",
	})

	AMQPPublishTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "amqp_publish_total",
		Help: "Total AMQP publish attempts grouped by routing key and result.",
	}, []string{"routing_key", "result"})

	AMQPPublishDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "amqp_publish_duration_seconds",
		Help:    "AMQP publish latency in seconds grouped by routing key.",
		Buckets: prometheus.DefBuckets,
	}, []string{"routing_key"})

	AMQPConnectionState = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "amqp_connection_state",
		Help: "AMQP connection state: 1 connected, 0 disconnected.",
	})

	AMQPReconnectTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "amqp_reconnect_total",
		Help: "Total AMQP reconnect attempts grouped by result.",
	}, []string{"result"})
)

var (
	lastDBAcquireDuration float64
	haveDBAcquireStat     bool
	dbStatsMu             sync.Mutex
	dbPoolRef             *pgxpool.Pool
)

func init() {
	prometheus.MustRegister(ActiveAuctions)
	prometheus.MustRegister(ManagedAuctions)
	prometheus.MustRegister(AuctionConnections)
	prometheus.MustRegister(AuctionsByStatus)
	prometheus.MustRegister(MemoryAlloc)
	prometheus.MustRegister(MemorySys)
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPInFlight)
	prometheus.MustRegister(WSConnectionsActive)
	prometheus.MustRegister(WSConnectTotal)
	prometheus.MustRegister(WSMessagesTotal)
	prometheus.MustRegister(WSMessageDuration)
	prometheus.MustRegister(AuctionBidsTotal)
	prometheus.MustRegister(AuctionBidAmount)
	prometheus.MustRegister(SchedulerScansTotal)
	prometheus.MustRegister(SchedulerFoundAuctions)
	prometheus.MustRegister(SchedulerActivationTotal)
	prometheus.MustRegister(RecoveryTotal)
	prometheus.MustRegister(DBQueryTotal)
	prometheus.MustRegister(DBQueryDuration)
	prometheus.MustRegister(DBPoolAcquired)
	prometheus.MustRegister(DBPoolIdle)
	prometheus.MustRegister(DBPoolTotal)
	prometheus.MustRegister(DBPoolAcquireWaitSeconds)
	prometheus.MustRegister(AMQPPublishTotal)
	prometheus.MustRegister(AMQPPublishDuration)
	prometheus.MustRegister(AMQPConnectionState)
	prometheus.MustRegister(AMQPReconnectTotal)
}

func Collect(manager *auction.Manager) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	MemoryAlloc.Set(float64(m.Alloc))
	MemorySys.Set(float64(m.Sys))

	sessions := manager.Sessions()

	var (
		activeCount     int
		scheduledCount  int
		finishedCount   int
		connectionCount int
	)

	for _, session := range sessions {
		switch session.Status() {
		case auction.StatusScheduled:
			scheduledCount++
		case auction.StatusActive:
			activeCount++
		case auction.StatusFinished:
			finishedCount++
		}
		connectionCount += session.ConnectionsCount()
	}

	ManagedAuctions.Set(float64(len(sessions)))
	ActiveAuctions.Set(float64(activeCount))
	AuctionConnections.Set(float64(connectionCount))
	AuctionsByStatus.WithLabelValues(string(auction.StatusScheduled)).Set(float64(scheduledCount))
	AuctionsByStatus.WithLabelValues(string(auction.StatusActive)).Set(float64(activeCount))
	AuctionsByStatus.WithLabelValues(string(auction.StatusFinished)).Set(float64(finishedCount))

	dbStatsMu.Lock()
	pool := dbPoolRef
	dbStatsMu.Unlock()
	CollectDBPool(pool)
}

func SetDBPool(pool *pgxpool.Pool) {
	dbStatsMu.Lock()
	defer dbStatsMu.Unlock()
	dbPoolRef = pool
}

func CollectDBPool(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}

	stats := pool.Stat()
	DBPoolAcquired.Set(float64(stats.AcquiredConns()))
	DBPoolIdle.Set(float64(stats.IdleConns()))
	DBPoolTotal.Set(float64(stats.TotalConns()))

	dbStatsMu.Lock()
	defer dbStatsMu.Unlock()

	current := stats.AcquireDuration().Seconds()
	if !haveDBAcquireStat {
		lastDBAcquireDuration = current
		haveDBAcquireStat = true
		return
	}
	if current > lastDBAcquireDuration {
		DBPoolAcquireWaitSeconds.Add(current - lastDBAcquireDuration)
	}
	lastDBAcquireDuration = current
}

func ObserveDBQuery(op string, started time.Time, err error) {
	DBQueryDuration.WithLabelValues(op).Observe(time.Since(started).Seconds())
	if err != nil {
		DBQueryTotal.WithLabelValues(op, "error").Inc()
		return
	}
	DBQueryTotal.WithLabelValues(op, "ok").Inc()
}

func ObserveAMQPPublish(routingKey string, started time.Time, err error) {
	AMQPPublishDuration.WithLabelValues(routingKey).Observe(time.Since(started).Seconds())
	if err != nil {
		AMQPPublishTotal.WithLabelValues(routingKey, "error").Inc()
		return
	}
	AMQPPublishTotal.WithLabelValues(routingKey, "ok").Inc()
}

func RecordAMQPConnectionState(connected bool) {
	if connected {
		AMQPConnectionState.Set(1)
		return
	}
	AMQPConnectionState.Set(0)
}

func ObserveWSMessage(msgType string, started time.Time, err error) {
	WSMessageDuration.WithLabelValues(msgType).Observe(time.Since(started).Seconds())
	if err != nil {
		WSMessagesTotal.WithLabelValues("in", msgType, "error").Inc()
		return
	}
	WSMessagesTotal.WithLabelValues("in", msgType, "ok").Inc()
}

func ObserveBidResult(result auction.BidResult, bidAmount int64) {
	if result.Accepted {
		AuctionBidsTotal.WithLabelValues("accepted", "none").Inc()
		AuctionBidAmount.Observe(float64(bidAmount))
		return
	}
	AuctionBidsTotal.WithLabelValues("rejected", normalizeBidReason(result.Error)).Inc()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HTTPInFlight.Inc()
		defer HTTPInFlight.Dec()

		started := time.Now()
		ww := &statusCapturingWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		route := routePattern(r)
		status := strconv.Itoa(ww.statusCode)
		method := strings.ToUpper(r.Method)
		HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, route, status).Observe(time.Since(started).Seconds())
	})
}

type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "unknown"
	}
	pattern := rctx.RoutePattern()
	if pattern == "" {
		return "unknown"
	}
	return pattern
}

func normalizeBidReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "":
		return "unknown"
	case auction.ErrNotActive.Error():
		return "not_active"
	case auction.ErrFinished.Error():
		return "finished"
	case auction.ErrRateLimited.Error():
		return "rate_limited"
	case auction.ErrBidNotLower.Error():
		return "not_lower"
	case auction.ErrBidNotAligned.Error():
		return "not_aligned"
	case "missing companyID or personID", "missing company_id or person_id":
		return "invalid_identity"
	case "failed to persist bid":
		return "persist_error"
	default:
		return "other"
	}
}
