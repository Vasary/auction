package http

import (
	"auction-core/internal/auction"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandler_DeleteAuction(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return &auction.PersistedAuction{
					TenderID: tenderID,
					Status:   auction.StatusScheduled,
				}, nil
			},
			onDelete: func(ctx context.Context, tenderID uuid.UUID) error {
				return nil
			},
		}
		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		req := httptest.NewRequest(http.MethodDelete, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.DeleteAuction(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("cannot delete started", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return &auction.PersistedAuction{
					TenderID: tenderID,
					Status:   auction.StatusActive,
				}, nil
			},
		}
		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		req := httptest.NewRequest(http.MethodDelete, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.DeleteAuction(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}
