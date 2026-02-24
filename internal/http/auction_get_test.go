package http

import (
	"auction-core/internal/auction"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_GetAuction(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return &auction.PersistedAuction{
					TenderID:   tenderID,
					StartPrice: 1000,
					Status:     auction.StatusActive,
				}, nil
			},
		}

		mockPartRepo := &mockParticipantRepo{
			onListParticipants: func(ctx context.Context, tenderID uuid.UUID) ([]uuid.UUID, error) {
				return []uuid.UUID{uuid.MustParse("d24a2778-d1c3-4acf-a771-755ab3cdab4d")}, nil
			},
		}

		h := NewHandler(nil, mockPartRepo, mockRepo, nil, logger)

		req := httptest.NewRequest(http.MethodGet, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.GetAuction(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp getAuctionResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), resp.TenderID)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return nil, auction.ErrNotFound
			},
		}

		h := NewHandler(nil, &mockParticipantRepo{}, mockRepo, nil, logger)

		req := httptest.NewRequest(http.MethodGet, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.GetAuction(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
