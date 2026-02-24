package auction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestManager_LifeCycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockRepo{}
	pub := &mockPublisher{}
	manager := NewManager(repo, pub, logger)

	now := time.Now()
	tenderID := uuid.MustParse("00000000-0000-0000-0000-000000000501")
	tenderDeleteID := uuid.MustParse("00000000-0000-0000-0000-000000000999")
	cfg := Config{
		TenderID:   tenderID,
		StartPrice: 1000,
		Step:       10,
		StartAt:    now.Add(1 * time.Hour),
		EndAt:      now.Add(2 * time.Hour),
	}

	s, err := manager.Create(cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if s == nil {
		t.Fatal("session is nil")
	}

	s2, ok := manager.Get(tenderID)
	if !ok || s2 != s {
		t.Error("failed to get created session")
	}

	sessions := manager.Sessions()
	if len(sessions) != 1 || sessions[tenderID] != s {
		t.Error("Sessions() returned incorrect map")
	}

	err = manager.Update(context.Background(), tenderID, func(a *PersistedAuction) {
		a.StartPrice = 2000
	})
	if err != ErrCannotUpdateStarted {
		t.Errorf("expected ErrCannotUpdateStarted, got %v", err)
	}

	cfgScheduled := Config{
		TenderID:   tenderDeleteID,
		StartPrice: 1000,
		Step:       10,
		StartAt:    now.Add(24 * time.Hour),
		EndAt:      now.Add(25 * time.Hour),
	}
	manager.Create(cfgScheduled)

	time.Sleep(10 * time.Millisecond)

	err = manager.Delete(context.Background(), tenderDeleteID)
	if err != nil {
		t.Errorf("failed to delete scheduled session: %v", err)
	}

	_, ok = manager.Get(tenderDeleteID)
	if ok {
		t.Error("scheduled session still exists after delete")
	}
}

func TestManager_RecoverSessions(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockRepo{}
	pub := &mockPublisher{}
	manager := NewManager(repo, pub, logger)

	err := manager.RecoverSessions(context.Background(), repo, nil, 5*time.Minute)
	if err != nil {
		t.Errorf("RecoverSessions failed: %v", err)
	}
}
