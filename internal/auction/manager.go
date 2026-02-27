package auction

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Manager struct {
	mu                 sync.RWMutex
	auctionsRepository AuctionsRepository
	eventPublisher     EventPublisher
	sessions           map[uuid.UUID]*Session
	logger             *zap.Logger
}

func NewManager(auctionsRepository AuctionsRepository, eventPublisher EventPublisher, logger *zap.Logger) *Manager {
	return &Manager{
		auctionsRepository: auctionsRepository,
		eventPublisher:     eventPublisher,
		sessions:           make(map[uuid.UUID]*Session),
		logger:             logger,
	}
}

func (m *Manager) Create(cfg Config) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[cfg.TenderID]; ok {
		m.logger.Debug("session already exists",
			zap.String("tender_id", cfg.TenderID.String()),
		)
		return s, nil
	}

	m.logger.Info("creating auction session",
		zap.String("tender_id", cfg.TenderID.String()),
	)

	s, err := NewSession(cfg, m.auctionsRepository, m.eventPublisher, m.logger)
	if err != nil {
		m.logger.Error("failed to create session",
			zap.String("tender_id", cfg.TenderID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	m.sessions[cfg.TenderID] = s
	s.Start()

	m.logger.Info("auction session started",
		zap.String("tender_id", cfg.TenderID.String()),
	)

	return s, nil
}

func (m *Manager) Get(tenderID uuid.UUID) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[tenderID]
	return s, ok
}

func (m *Manager) Delete(ctx context.Context, tenderID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[tenderID]; ok {
		if s.Status() != StatusScheduled {
			return ErrCannotDeleteStarted
		}
		s.Stop()
		delete(m.sessions, tenderID)
	}

	a, err := m.auctionsRepository.GetByID(ctx, tenderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	if a.Status != StatusScheduled {
		return ErrCannotDeleteStarted
	}

	return m.auctionsRepository.Delete(ctx, tenderID)
}

func (r *Manager) Update(ctx context.Context, tenderID uuid.UUID, updateFn func(*PersistedAuction)) error {
	r.mu.RLock()
	_, exists := r.sessions[tenderID]
	r.mu.RUnlock()

	if exists {
		return ErrCannotUpdateStarted
	}

	a, err := r.auctionsRepository.GetByID(ctx, tenderID)
	if err != nil {
		return err
	}

	if a.Status == StatusFinished {
		return ErrCannotUpdateFinished
	}

	if a.Status != StatusScheduled {
		return ErrCannotUpdateStarted
	}

	updateFn(a)

	now := time.Now()
	if !a.EndAt.After(a.StartAt) {
		return ErrInvalidConfig
	}
	if a.EndAt.Before(now) {
		return ErrInvalidConfig
	}

	return r.auctionsRepository.Update(ctx, a)
}

func (m *Manager) Sessions() map[uuid.UUID]*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make(map[uuid.UUID]*Session, len(m.sessions))
	for k, v := range m.sessions {
		sessions[k] = v
	}
	return sessions
}

func (m *Manager) RecoverSessions(
	ctx context.Context,
	auctionsRepository AuctionsRepository,
	bidsRepository BidRepository,
	window time.Duration,
) error {
	now := time.Now()
	from := time.Time{}
	to := now.Add(100 * 365 * 24 * time.Hour)

	auctions, err := auctionsRepository.FindStartingBetween(ctx, from, to)
	if err != nil {
		return err
	}

	var (
		resumed  int
		finished int
		skipped  int
	)

	for _, a := range auctions {
		if a.Status != StatusScheduled && a.Status != StatusActive {
			skipped++
			continue
		}

		if now.After(a.EndAt) {
			if err := auctionsRepository.UpdateStatus(ctx, a.TenderID, StatusFinished); err != nil {
				m.logger.Error("failed to finalize auction during recovery",
					zap.String("tender_id", a.TenderID.String()),
					zap.Error(err),
				)
				continue
			}
			finished++
			continue
		}

		if !a.EndAt.After(a.StartAt) {
			m.logger.Warn("skipping auction with invalid time configuration",
				zap.String("tender_id", a.TenderID.String()),
				zap.Time("start_at", a.StartAt),
				zap.Time("end_at", a.EndAt),
			)
			skipped++
			continue
		}

		cfg := Config{
			TenderID:           a.TenderID,
			StartPrice:         a.StartPrice,
			CurrentPrice:       a.CurrentPrice,
			Step:               a.Step,
			StartAt:            a.StartAt,
			EndAt:              a.EndAt,
			RateLimitPerBidder: 100 * time.Millisecond,
			BroadcastBuffer:    64,
		}

		if cfg.CurrentPrice == 0 {
			cfg.CurrentPrice = a.StartPrice
		}

		if a.WinnerBidID != nil {
			b, err := bidsRepository.GetByID(ctx, *a.WinnerBidID)
			if err != nil {
				m.logger.Error("failed to load winning bid",
					zap.String("tender_id", a.TenderID.String()),
					zap.Error(err),
				)
				skipped++
				continue
			}

			if a.WinnerID != nil {
				cfg.WinnerID = a.WinnerID
			}

			cfg.CurrentPrice = b.BidAmount
			cfg.LatestBid = LatestBid{
				ID:        b.ID,
				CompanyID: b.CompanyID,
				PersonID:  b.PersonID,
				BidAmount: b.BidAmount,
				BidAt:     b.CreatedAt,
			}

		}

		_, err := m.Create(cfg)
		if err == nil {
			resumed++
		} else {
			skipped++
		}
	}

	m.logger.Info("auction recovery completed",
		zap.Int("loaded", len(auctions)),
		zap.Int("resumed_sessions", resumed),
		zap.Int("marked_finished", finished),
		zap.Int("skipped", skipped),
		zap.Time("now", now),
		zap.Duration("window", window),
	)

	return nil
}
