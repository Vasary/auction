package http

import (
	"auction-core/internal/auction"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_ListAuctions(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onList: func(ctx context.Context) ([]auction.PersistedAuction, error) {
				return []auction.PersistedAuction{
					{TenderID: uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), StartPrice: 1000},
					{TenderID: uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"), StartPrice: 2000},
				}, nil
			},
		}

		h := NewHandler(nil, nil, mockRepo, nil, logger)

		req := httptest.NewRequest(http.MethodGet, "/auctions", nil)
		w := httptest.NewRecorder()

		h.ListAuctions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []auctionListItem
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), resp[0].TenderID)
		assert.Equal(t, uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"), resp[1].TenderID)
	})
}
