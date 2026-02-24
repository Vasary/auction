package http

import (
	"auction-core/internal/auction"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_UpdateAuction(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{}
		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		mockRepo.onGetByID = func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
			now := time.Now()
			return &auction.PersistedAuction{
				TenderID: tenderID,
				Status:   auction.StatusScheduled,
				StartAt:  now,
				EndAt:    now.Add(time.Hour),
			}, nil
		}
		mockRepo.onUpdate = func(ctx context.Context, a *auction.PersistedAuction) error {
			assert.Equal(t, int64(1500), a.StartPrice)
			return nil
		}

		newPrice := int64(1500)
		reqBody := updateAuctionRequest{
			StartPrice: &newPrice,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.UpdateAuction(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "updated", resp["status"])
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return nil, auction.ErrNotFound
			},
		}
		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		reqBody := updateAuctionRequest{}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.UpdateAuction(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid config - endAt before startAt", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{}
		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		mockRepo.onGetByID = func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
			now := time.Now()
			return &auction.PersistedAuction{
				TenderID: tenderID,
				Status:   auction.StatusScheduled,
				StartAt:  now,
				EndAt:    now.Add(time.Hour),
			}, nil
		}

		badEndAt := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		reqBody := updateAuctionRequest{
			EndAt: &badEndAt,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.UpdateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid auction configuration (check dates)")
	})
}
