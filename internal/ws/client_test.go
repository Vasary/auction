package ws

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auction-core/internal/auction"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type mockParticipantRepo struct{}

func (m *mockParticipantRepo) AddParticipant(ctx context.Context, tenderID, companyID uuid.UUID) error {
	return nil
}
func (m *mockParticipantRepo) IsParticipant(ctx context.Context, tenderID, companyID uuid.UUID) (bool, error) {
	if companyID == uuid.MustParse("d53c7979-72b8-4bdf-bf52-c93f29916c1a") {
		return false, nil
	}
	return true, nil
}
func (m *mockParticipantRepo) ListParticipants(ctx context.Context, tenderID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

type mockAuctionRepo struct{}

func (m *mockAuctionRepo) Create(ctx context.Context, a *auction.PersistedAuction) error { return nil }
func (m *mockAuctionRepo) GetByID(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
	return nil, nil
}
func (m *mockAuctionRepo) Update(ctx context.Context, a *auction.PersistedAuction) error { return nil }
func (m *mockAuctionRepo) UpdateStatus(ctx context.Context, tenderID uuid.UUID, status auction.Status) error {
	return nil
}
func (m *mockAuctionRepo) Delete(ctx context.Context, tenderID uuid.UUID) error { return nil }
func (m *mockAuctionRepo) FindStartingBetween(ctx context.Context, from, to time.Time) ([]auction.PersistedAuction, error) {
	return nil, nil
}
func (m *mockAuctionRepo) List(ctx context.Context) ([]auction.PersistedAuction, error) {
	return nil, nil
}
func (m *mockAuctionRepo) CreateBidTx(ctx context.Context, tenderID, companyID, personID uuid.UUID, amount int64) error {
	return nil
}

func TestWebSocketClient(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockAuctionRepo{}
	partRepo := &mockParticipantRepo{}

	cfg := auction.Config{
		TenderID:        uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
		StartPrice:      1000,
		CurrentPrice:    1000,
		Step:            10,
		StartAt:         time.Now().Add(-1 * time.Hour),
		EndAt:           time.Now().Add(1 * time.Hour),
		BroadcastBuffer: 1024,
	}
	session, _ := auction.NewSession(cfg, repo, nil, logger)
	session.Start()
	defer session.Stop()

	var (
		server *httptest.Server
		errSrv error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				errSrv = fmt.Errorf("cannot start httptest server: %v", r)
			}
		}()
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			client := NewClient(conn, session, partRepo, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), logger)
			go client.Run()
		}))
	}()
	if errSrv != nil {
		t.Skip(errSrv.Error())
	}
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer ws.Close()

	var ev auction.Event
	err = ws.ReadJSON(&ev)
	if err != nil {
		t.Fatalf("failed to read initial snapshot: %v", err)
	}
	if ev.Type != auction.EventSnapshot {
		t.Errorf("expected EventSnapshot, got %v", ev.Type)
	}

	bidMsg := incomingMessage{
		Type:      "place_bid",
		Bid:       990,
		CompanyID: uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"),
		PersonID:  uuid.MustParse("c35a2778-d1c3-4acf-a771-755ab3cdab4d"),
	}
	err = ws.WriteJSON(bidMsg)
	if err != nil {
		t.Fatalf("failed to write bid: %v", err)
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		var msg map[string]any
		readDone := make(chan error, 1)
		go func() {
			readDone <- ws.ReadJSON(&msg)
		}()

		select {
		case err := <-readDone:
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}
			if msg["type"] == string(auction.EventPriceUpdated) {
				return
			}
		case <-timer.C:
			t.Fatalf("expected EventPriceUpdated")
		}
	}
}

func TestWebSocketClientRejectsUnregisteredParticipant(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockAuctionRepo{}
	partRepo := &mockParticipantRepo{}

	cfg := auction.Config{
		TenderID:        uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
		StartPrice:      1000,
		CurrentPrice:    1000,
		Step:            10,
		StartAt:         time.Now().Add(-1 * time.Hour),
		EndAt:           time.Now().Add(1 * time.Hour),
		BroadcastBuffer: 1024,
	}
	session, _ := auction.NewSession(cfg, repo, nil, logger)
	session.Start()
	defer session.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := NewClient(conn, session, partRepo, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), logger)
		go client.Run()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer ws.Close()

	var snapshot auction.Event
	if err := ws.ReadJSON(&snapshot); err != nil {
		t.Fatalf("failed to read initial snapshot: %v", err)
	}

	bidMsg := incomingMessage{
		Type:      "place_bid",
		Bid:       990,
		CompanyID: uuid.MustParse("d53c7979-72b8-4bdf-bf52-c93f29916c1a"),
		PersonID:  uuid.MustParse("6179ac4e-b0c4-44ed-8b18-a616a16fb936"),
	}
	if err := ws.WriteJSON(bidMsg); err != nil {
		t.Fatalf("failed to write bid: %v", err)
	}

	var result auction.BidResult
	if err := ws.ReadJSON(&result); err != nil {
		t.Fatalf("failed to read bid result: %v", err)
	}

	if result.Accepted {
		t.Fatalf("expected bid to be rejected for unregistered participant")
	}

	if result.Error != "bid rejected: you are not registered as auction participant" {
		t.Fatalf("unexpected error message: %q", result.Error)
	}
}
