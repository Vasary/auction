package http

import (
	"auction-core/internal/auction"
	"context"
	"time"

	"github.com/google/uuid"
)

type mockAuctionRepo struct {
	auction.AuctionsRepository
	onCreate              func(ctx context.Context, a *auction.PersistedAuction) error
	onGetByID             func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error)
	onList                func(ctx context.Context) ([]auction.PersistedAuction, error)
	onUpdate              func(ctx context.Context, a *auction.PersistedAuction) error
	onDelete              func(ctx context.Context, tenderID uuid.UUID) error
	onFindStartingBetween func(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error)
}

func (m *mockAuctionRepo) Create(ctx context.Context, a *auction.PersistedAuction) error {
	if m.onCreate != nil {
		return m.onCreate(ctx, a)
	}
	return nil
}

func (m *mockAuctionRepo) GetByID(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
	if m.onGetByID != nil {
		return m.onGetByID(ctx, tenderID)
	}
	return nil, nil
}

func (m *mockAuctionRepo) List(ctx context.Context) ([]auction.PersistedAuction, error) {
	if m.onList != nil {
		return m.onList(ctx)
	}
	return nil, nil
}

func (m *mockAuctionRepo) Update(ctx context.Context, a *auction.PersistedAuction) error {
	if m.onUpdate != nil {
		return m.onUpdate(ctx, a)
	}
	return nil
}

func (m *mockAuctionRepo) Delete(ctx context.Context, tenderID uuid.UUID) error {
	if m.onDelete != nil {
		return m.onDelete(ctx, tenderID)
	}
	return nil
}

func (m *mockAuctionRepo) FindStartingBetween(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
	if m.onFindStartingBetween != nil {
		return m.onFindStartingBetween(ctx, from, to)
	}
	return nil, nil
}

type mockParticipantRepo struct {
	auction.ParticipantRepository
	onAddParticipant   func(ctx context.Context, tenderID, companyID uuid.UUID) error
	onIsParticipant    func(ctx context.Context, tenderID, companyID uuid.UUID) (bool, error)
	onListParticipants func(ctx context.Context, tenderID uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockParticipantRepo) AddParticipant(ctx context.Context, tenderID, companyID uuid.UUID) error {
	if m.onAddParticipant != nil {
		return m.onAddParticipant(ctx, tenderID, companyID)
	}
	return nil
}

func (m *mockParticipantRepo) IsParticipant(ctx context.Context, tenderID, companyID uuid.UUID) (bool, error) {
	if m.onIsParticipant != nil {
		return m.onIsParticipant(ctx, tenderID, companyID)
	}
	return false, nil
}
func (m *mockParticipantRepo) ListParticipants(ctx context.Context, tenderID uuid.UUID) ([]uuid.UUID, error) {
	if m.onListParticipants != nil {
		return m.onListParticipants(ctx, tenderID)
	}
	return []uuid.UUID{}, nil
}

type mockBidRepo struct {
	auction.BidRepository
	onListBids func(ctx context.Context, tenderID uuid.UUID) ([]auction.Bid, error)
}

func (m *mockBidRepo) ListBids(ctx context.Context, tenderID uuid.UUID) ([]auction.Bid, error) {
	if m.onListBids != nil {
		return m.onListBids(ctx, tenderID)
	}
	return nil, nil
}
