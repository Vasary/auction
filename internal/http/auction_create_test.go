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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_CreateAuction(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onCreate: func(ctx context.Context, a *auction.PersistedAuction) error {
				assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), a.TenderID)
				assert.Equal(t, int64(1000), a.StartPrice)
				return nil
			},
		}

		manager := auction.NewManager(mockRepo, nil, logger)
		h := NewHandler(manager, nil, mockRepo, nil, logger)

		reqBody := createAuctionRequest{
			TenderID:   uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
			StartPrice: 1000,
			Step:       10,
			StartAt:    time.Now().Add(time.Hour).Format(time.RFC3339),
			EndAt:      time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			CreatedBy:  uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp createAuctionResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), resp.TenderID)
		assert.Equal(t, auction.StatusScheduled, resp.Status)
	})

	t.Run("invalid input", func(t *testing.T) {
		h := NewHandler(nil, nil, nil, nil, logger)

		reqBody := createAuctionRequest{
			TenderID: uuid.Nil,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("already exists", func(t *testing.T) {
		mockRepo := &mockAuctionRepo{
			onCreate: func(ctx context.Context, a *auction.PersistedAuction) error {
				return auction.ErrAlreadyExists
			},
		}

		h := NewHandler(nil, nil, mockRepo, nil, logger)

		reqBody := createAuctionRequest{
			TenderID:   uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
			StartPrice: 1000,
			Step:       10,
			StartAt:    time.Now().Add(time.Hour).Format(time.RFC3339),
			EndAt:      time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			CreatedBy:  uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("endAt before startAt", func(t *testing.T) {
		h := NewHandler(nil, nil, nil, nil, logger)

		reqBody := createAuctionRequest{
			TenderID:   uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
			StartPrice: 1000,
			Step:       10,
			StartAt:    time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			EndAt:      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			CreatedBy:  uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "endAt must be after startAt")
	})

	t.Run("endAt in the past", func(t *testing.T) {
		h := NewHandler(nil, nil, nil, nil, logger)

		reqBody := createAuctionRequest{
			TenderID:   uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
			StartPrice: 1000,
			Step:       10,
			StartAt:    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			EndAt:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			CreatedBy:  uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "endAt cannot be in the past")
	})
}
