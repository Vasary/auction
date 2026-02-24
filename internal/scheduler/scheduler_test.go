package scheduler

import (
	"auction-core/internal/auction"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockRepository struct {
	onFindStartingBetween func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error)
}

func (m *mockRepository) FindStartingBetween(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
	if m.onFindStartingBetween != nil {
		return m.onFindStartingBetween(ctx, from, to)
	}
	return nil, nil
}

func TestScheduler_Scan(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepository{}
	manager := auction.NewManager(nil, nil, logger)
	s := &Scheduler{
		Manager:    manager,
		Repository: repo,
		Interval:   time.Minute,
		Logger:     logger,
	}

	t.Run("should activate new auctions", func(t *testing.T) {
		now := time.Now()
		tenderID := uuid.MustParse("00000000-0000-0000-0000-000000000601")
		auctions := []auction.PersistedAuction{
			{
				TenderID:     tenderID,
				StartPrice:   1000,
				CurrentPrice: 1000,
				Step:         100,
				StartAt:      now.Add(2 * time.Minute),
				EndAt:        now.Add(10 * time.Minute),
			},
		}

		repo.onFindStartingBetween = func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
			return auctions, nil
		}

		s.scan(context.Background())

		_, exists := manager.Get(tenderID)
		assert.True(t, exists)
	})

	t.Run("should not reactivate existing sessions", func(t *testing.T) {
		now := time.Now()
		tenderID := uuid.MustParse("00000000-0000-0000-0000-000000000602")

		_, err := manager.Create(auction.Config{
			TenderID:     tenderID,
			StartPrice:   1000,
			CurrentPrice: 1000,
			Step:         100,
			StartAt:      now.Add(time.Minute),
			EndAt:        now.Add(10 * time.Minute),
		})
		require.NoError(t, err)

		auctions := []auction.PersistedAuction{
			{
				TenderID:     tenderID,
				StartPrice:   1000,
				CurrentPrice: 1000,
				Step:         100,
				StartAt:      now.Add(2 * time.Minute),
				EndAt:        now.Add(10 * time.Minute),
			},
		}

		repo.onFindStartingBetween = func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
			return auctions, nil
		}

		s.scan(context.Background())

		_, exists := manager.Get(tenderID)
		assert.True(t, exists)
	})

	t.Run("should handle repository error", func(t *testing.T) {
		repo.onFindStartingBetween = func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
			return nil, errors.New("db error")
		}

		s.scan(context.Background())
	})
}

func TestScheduler_StartStop(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepository{}
	manager := auction.NewManager(nil, nil, logger)

	s := &Scheduler{
		Manager:    manager,
		Repository: repo,
		Interval:   10 * time.Millisecond,
		Logger:     logger,
	}

	repo.onFindStartingBetween = func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	s.Start(ctx)
}
