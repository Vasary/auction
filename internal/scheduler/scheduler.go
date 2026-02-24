package scheduler

import (
	"context"
	"time"

	"auction-core/internal/metrics"
	"go.uber.org/zap"

	"auction-core/internal/auction"
)

type Scheduler struct {
	Manager    *auction.Manager
	Repository interface {
		FindStartingBetween(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error)
	}
	Interval time.Duration
	Logger   *zap.Logger
}

func (s *Scheduler) Start(ctx context.Context) {
	if s.Logger != nil {
		s.Logger.Info("auction scheduler started",
			zap.Duration("interval", s.Interval),
			zap.Duration("activation_window", 5*time.Minute),
		)
	}

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if s.Logger != nil {
				s.Logger.Info("auction scheduler stopped")
			}
			return

		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *Scheduler) scan(ctx context.Context) {
	now := time.Now()
	until := now.Add(5 * time.Minute)

	auctions, err := s.Repository.FindStartingBetween(ctx, now, until)
	if err != nil {
		metrics.SchedulerScansTotal.WithLabelValues("error").Inc()
		if s.Logger != nil {
			s.Logger.Error("scheduler scan failed", zap.Error(err))
		}
		return
	}
	metrics.SchedulerScansTotal.WithLabelValues("ok").Inc()
	metrics.SchedulerFoundAuctions.Observe(float64(len(auctions)))

	activated := 0

	for _, a := range auctions {

		if _, exists := s.Manager.Get(a.TenderID); exists {
			metrics.SchedulerActivationTotal.WithLabelValues("already_loaded").Inc()
			continue
		}

		cfg := auction.Config{
			TenderID:           a.TenderID,
			StartPrice:         a.StartPrice,
			CurrentPrice:       a.CurrentPrice,
			Step:               a.Step,
			StartAt:            a.StartAt,
			EndAt:              a.EndAt,
			RateLimitPerBidder: 500 * time.Millisecond,
			BroadcastBuffer:    64,
		}

		if _, err := s.Manager.Create(cfg); err == nil {
			activated++
			metrics.SchedulerActivationTotal.WithLabelValues("activated").Inc()
		} else {
			metrics.SchedulerActivationTotal.WithLabelValues("error").Inc()
		}
	}

	if s.Logger != nil {
		s.Logger.Info("scheduler scan completed",
			zap.Int("found", len(auctions)),
			zap.Int("activated", activated),
			zap.Time("from", now),
			zap.Time("to", until),
		)
	}
}
