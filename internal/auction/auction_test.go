package auction

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func testUUID(v string) uuid.UUID {
	return uuid.MustParse(v)
}

type mockRepo struct {
	auctions []PersistedAuction
}

func (m *mockRepo) Create(ctx context.Context, a *PersistedAuction) error { return nil }
func (m *mockRepo) GetByID(ctx context.Context, tenderID uuid.UUID) (*PersistedAuction, error) {
	for _, a := range m.auctions {
		if a.TenderID == tenderID {
			return &a, nil
		}
	}
	if tenderID == testUUID("00000000-0000-0000-0000-000000000999") {
		return nil, ErrNotFound
	}
	return &PersistedAuction{
		TenderID:   tenderID,
		StartPrice: 1000,
		Status:     StatusActive,
	}, nil
}
func (m *mockRepo) Update(ctx context.Context, a *PersistedAuction) error { return nil }
func (m *mockRepo) UpdateStatus(ctx context.Context, tenderID uuid.UUID, status Status) error {
	return nil
}
func (m *mockRepo) Delete(ctx context.Context, tenderID uuid.UUID) error { return nil }
func (m *mockRepo) FindStartingBetween(ctx context.Context, from, to time.Time) ([]PersistedAuction, error) {
	return m.auctions, nil
}

type mockBidRepo struct {
	bids map[int64]Bid
}

func (m *mockBidRepo) GetByID(ctx context.Context, id int64) (Bid, error) {
	if b, ok := m.bids[id]; ok {
		return b, nil
	}
	return Bid{}, errors.New("not found")
}

func (m *mockBidRepo) ListBids(ctx context.Context, tenderID uuid.UUID) ([]Bid, error) {
	return nil, nil
}

func TestRecoveryZeroCurrentPrice(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}
	manager := NewManager(repo, pub, logger)

	tenderID := testUUID("00000000-0000-0000-0000-000000000101")
	startAt := time.Now().Add(-1 * time.Hour)
	endAt := time.Now().Add(1 * time.Hour)

	repo.auctions = []PersistedAuction{
		{
			TenderID:     tenderID,
			StartPrice:   200000,
			CurrentPrice: 0,
			Status:       StatusActive,
			StartAt:      startAt,
			EndAt:        endAt,
			Step:         1500,
		},
	}

	ctx := context.Background()
	err := manager.RecoverSessions(ctx, repo, nil, 0)
	if err != nil {
		t.Fatalf("failed to recover: %v", err)
	}

	session, ok := manager.Get(tenderID)
	if !ok {
		t.Fatalf("session not found")
	}

	snap := session.Snapshot()
	if snap.CurrentPrice != 200000 {
		t.Errorf("expected current price %d, got %d", 200000, snap.CurrentPrice)
	}
}

func TestRecoveryWithWinner(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	bidRepo := &mockBidRepo{
		bids: make(map[int64]Bid),
	}
	pub := &mockPublisher{}
	manager := NewManager(repo, pub, logger)

	tenderID := testUUID("00000000-0000-0000-0000-000000000102")
	winnerID := testUUID("00000000-0000-0000-0000-000000000201")
	personID := testUUID("00000000-0000-0000-0000-000000000301")
	bidID := int64(123)
	bidAmount := int64(198500)

	bidRepo.bids[bidID] = Bid{
		ID:        bidID,
		TenderID:  tenderID,
		CompanyID: winnerID,
		PersonID:  personID,
		BidAmount: bidAmount,
	}

	repo.auctions = []PersistedAuction{
		{
			TenderID:     tenderID,
			StartPrice:   200000,
			CurrentPrice: bidAmount,
			Status:       StatusActive,
			WinnerID:     &winnerID,
			WinnerBidID:  &bidID,
			StartAt:      time.Now().Add(-1 * time.Hour),
			EndAt:        time.Now().Add(1 * time.Hour),
			Step:         1500,
		},
	}

	ctx := context.Background()
	err := manager.RecoverSessions(ctx, repo, bidRepo, 0)
	if err != nil {
		t.Fatalf("failed to recover: %v", err)
	}

	session, ok := manager.Get(tenderID)
	if !ok {
		t.Fatalf("session not found")
	}

	snap := session.Snapshot()
	if snap.CurrentPrice != bidAmount {
		t.Errorf("expected current price %d, got %d", bidAmount, snap.CurrentPrice)
	}
	if snap.WinnerID == nil || *snap.WinnerID != winnerID {
		t.Errorf("expected winner %s, got %v", winnerID, snap.WinnerID)
	}
}
func (m *mockRepo) List(ctx context.Context) ([]PersistedAuction, error) {
	return []PersistedAuction{}, nil
}
func (m *mockRepo) CreateBidTx(ctx context.Context, tenderID, companyID, personID uuid.UUID, amount int64) error {
	return nil
}

type mockPublisher struct{}

func (m *mockPublisher) PublishAuctionStarted(tenderID uuid.UUID, snapshot Snapshot) error {
	return nil
}
func (m *mockPublisher) PublishAuctionFinished(tenderID uuid.UUID, snapshot Snapshot) error {
	return nil
}

func TestAuctionSessionFlow(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}

	now := time.Now()
	cfg := Config{
		TenderID:     testUUID("00000000-0000-0000-0000-000000000401"),
		StartPrice:   1000,
		CurrentPrice: 1000,
		Step:         100,
		StartAt:      now.Add(-1 * time.Minute),
		EndAt:        now.Add(1 * time.Minute),
	}

	session, err := NewSession(cfg, repo, pub, logger)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	session.Start()

	events, unsub := session.Subscribe()
	defer unsub()

	select {
	case ev := <-events:
		if ev.Type != EventSnapshot {
			t.Errorf("expected EventSnapshot, got %v", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventSnapshot")
	}

	if session.Status() != StatusActive {
		t.Errorf("expected status Active, got %v", session.Status())
	}

	res := session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 900)
	if !res.Accepted {
		t.Errorf("expected bid to be accepted, got error: %s", res.Error)
	}

	select {
	case ev := <-events:
		if ev.Type != EventPriceUpdated {
			t.Errorf("expected EventPriceUpdated, got %v", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventPriceUpdated")
	}

	if session.currentPrice != 900 {
		t.Errorf("expected price 900, got %d", session.currentPrice)
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000202"), testUUID("00000000-0000-0000-0000-000000000302"), 950)
	if res.Accepted {
		t.Error("expected bid 950 to be rejected")
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000202"), testUUID("00000000-0000-0000-0000-000000000302"), 850)
	if res.Accepted {
		t.Error("expected bid 850 to be rejected (step is 100)")
	}

	session.Stop()

	select {
	case ev := <-events:
		if ev.Type != EventFinished {
			t.Errorf("expected EventFinished, got %v", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventFinished")
	}

	if session.Status() != StatusFinished {
		t.Errorf("expected status Finished, got %v", session.Status())
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 700)
	if res.Accepted {
		t.Error("expected bid after finish to be rejected")
	}

	select {
	case _, ok := <-events:
		if ok {
			t.Error("expected events channel to be closed")
		}
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for events channel to close")
	}
}

func TestAuctionTwoParticipants(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}

	now := time.Now()
	cfg := Config{
		TenderID:           testUUID("00000000-0000-0000-0000-000000000402"),
		StartPrice:         1000,
		CurrentPrice:       1000,
		Step:               10,
		StartAt:            now.Add(-1 * time.Minute),
		EndAt:              now.Add(1 * time.Minute),
		RateLimitPerBidder: 1 * time.Millisecond,
	}

	session, _ := NewSession(cfg, repo, pub, logger)
	session.Start()
	defer session.Stop()

	res := session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 990)
	if !res.Accepted {
		t.Fatalf("bid from comp1 should be accepted: %s", res.Error)
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000202"), testUUID("00000000-0000-0000-0000-000000000302"), 980)
	if !res.Accepted {
		t.Errorf("bid from comp2 should be accepted: %s", res.Error)
	}

	if session.currentPrice != 980 {
		t.Errorf("expected current price 980, got %d", session.currentPrice)
	}
	expectedWinner2 := testUUID("00000000-0000-0000-0000-000000000202")
	if session.winnerID == nil || *session.winnerID != expectedWinner2 {
		t.Errorf("expected winner comp2, got %v", session.winnerID)
	}

	time.Sleep(10 * time.Millisecond)
	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 970)
	if !res.Accepted {
		t.Errorf("bid from comp1 should be accepted again: %s", res.Error)
	}

	expectedWinner1 := testUUID("00000000-0000-0000-0000-000000000201")
	if session.winnerID == nil || *session.winnerID != expectedWinner1 {
		t.Errorf("expected winner comp1, got %v", session.winnerID)
	}
}

func TestAuctionRateLimit(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}

	now := time.Now()
	cfg := Config{
		TenderID:           testUUID("00000000-0000-0000-0000-000000000403"),
		StartPrice:         1000,
		CurrentPrice:       1000,
		Step:               10,
		StartAt:            now.Add(-1 * time.Minute),
		EndAt:              now.Add(1 * time.Minute),
		RateLimitPerBidder: 500 * time.Millisecond,
	}

	session, _ := NewSession(cfg, repo, pub, logger)
	session.Start()
	defer session.Stop()

	res := session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 990)
	if !res.Accepted {
		t.Fatalf("first bid should be accepted: %s", res.Error)
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 980)
	if res.Accepted {
		t.Error("second bid should be rate limited")
	}
	if res.Error != ErrRateLimited.Error() {
		t.Errorf("expected error %v, got %s", ErrRateLimited, res.Error)
	}

	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000202"), testUUID("00000000-0000-0000-0000-000000000302"), 970)
	if !res.Accepted {
		t.Errorf("bid from another company should be accepted: %s", res.Error)
	}

	time.Sleep(600 * time.Millisecond)
	res = session.PlaceBid(testUUID("00000000-0000-0000-0000-000000000201"), testUUID("00000000-0000-0000-0000-000000000301"), 960)
	if !res.Accepted {
		t.Errorf("bid after rate limit period should be accepted: %s", res.Error)
	}
}

func TestAuctionScheduledToActive(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}

	now := time.Now()
	cfg := Config{
		TenderID:     testUUID("00000000-0000-0000-0000-000000000404"),
		StartPrice:   1000,
		CurrentPrice: 1000,
		Step:         10,
		StartAt:      now.Add(500 * time.Millisecond),
		EndAt:        now.Add(2 * time.Second),
	}

	session, _ := NewSession(cfg, repo, pub, logger)
	session.Start()
	defer session.Stop()

	if session.Status() != StatusScheduled {
		t.Errorf("expected StatusScheduled, got %v", session.Status())
	}

	events, unsub := session.Subscribe()
	defer unsub()

	select {
	case ev := <-events:
		if ev.Type != EventSnapshot {
			t.Errorf("expected EventSnapshot, got %v", ev.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for initial snapshot")
	}

	select {
	case ev := <-events:
		if ev.Type != EventStarted {
			t.Errorf("expected EventStarted, got %v", ev.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for EventStarted")
	}

	if session.Status() != StatusActive {
		t.Errorf("expected StatusActive, got %v", session.Status())
	}
}
