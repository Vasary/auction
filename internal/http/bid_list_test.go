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

func TestHandler_ListBids(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockBidRepo := &mockBidRepo{
			onListBids: func(ctx context.Context, tenderID uuid.UUID) ([]auction.Bid, error) {
				assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), tenderID)
				return []auction.Bid{
					{ID: 1, CompanyID: uuid.MustParse("c24a2778-d1c3-4acf-a771-755ab3cdab4d"), BidAmount: 100},
					{ID: 2, CompanyID: uuid.MustParse("d24a2778-d1c3-4acf-a771-755ab3cdab4d"), BidAmount: 90},
				}, nil
			},
		}

		h := NewHandler(nil, nil, nil, mockBidRepo, logger)

		req := httptest.NewRequest(http.MethodGet, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d/bids", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.ListBids(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []bidResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, int64(1), resp[0].ID)
		assert.Equal(t, int64(2), resp[1].ID)
	})
}
