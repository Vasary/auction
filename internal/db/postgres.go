package db

import (
	"context"
	"time"

	"auction-core/internal/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func New(ctx context.Context, log *zap.Logger, dsn string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("dns parsing issue", zap.Error(err))

		return nil
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatal("pool initialization issue", zap.Error(err))

		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil
	}

	log.Info("database connected")
	metrics.SetDBPool(pool)

	return pool
}
