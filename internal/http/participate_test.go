package http

import (
	"auction-core/internal/auction"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandler_Participate(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockAuctionRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return &auction.PersistedAuction{Status: auction.StatusActive}, nil
			},
		}
		mockParticipantRepo := &mockParticipantRepo{
			onAddParticipant: func(ctx context.Context, tenderID, companyID uuid.UUID) error {
				assert.Equal(t, uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"), tenderID)
				assert.Equal(t, uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d"), companyID)
				return nil
			},
		}

		h := NewHandler(nil, mockParticipantRepo, mockAuctionRepo, nil, logger)

		reqBody := participateRequest{CompanyID: uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d")}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d/participate", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.Participate(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("auction finished", func(t *testing.T) {
		mockAuctionRepo := &mockAuctionRepo{
			onGetByID: func(ctx context.Context, tenderID uuid.UUID) (*auction.PersistedAuction, error) {
				return &auction.PersistedAuction{Status: auction.StatusFinished}, nil
			},
		}

		h := NewHandler(nil, nil, mockAuctionRepo, nil, logger)

		reqBody := participateRequest{CompanyID: uuid.MustParse("b24a2778-d1c3-4acf-a771-755ab3cdab4d")}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d/participate", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.Participate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
