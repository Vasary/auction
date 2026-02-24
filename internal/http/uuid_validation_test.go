package http

import (
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
	"go.uber.org/zap"
)

func TestHandler_UUIDValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &mockAuctionRepo{}
	h := NewHandler(nil, nil, mockRepo, nil, logger)

	t.Run("CreateAuction invalid tenderId UUID", func(t *testing.T) {
		body := []byte(`{"tenderId":"not-a-uuid","startPrice":1000,"step":10,"startAt":"` + time.Now().Add(time.Hour).Format(time.RFC3339) + `","endAt":"` + time.Now().Add(2*time.Hour).Format(time.RFC3339) + `","createdBy":"e13a2778-d1c3-4acf-a771-755ab3cdab4d"}`)
		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid json")
	})

	t.Run("CreateAuction invalid createdBy UUID", func(t *testing.T) {
		body := []byte(`{"tenderId":"e13a2778-d1c3-4acf-a771-755ab3cdab4d","startPrice":1000,"step":10,"startAt":"` + time.Now().Add(time.Hour).Format(time.RFC3339) + `","endAt":"` + time.Now().Add(2*time.Hour).Format(time.RFC3339) + `","createdBy":"not-a-uuid"}`)
		req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.CreateAuction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid json")
	})

	t.Run("Participate invalid companyId UUID", func(t *testing.T) {
		mockParticipantRepo := &mockParticipantRepo{}
		h.ParticipantRepo = mockParticipantRepo
		body := []byte(`{"companyId":"not-a-uuid"}`)
		req := httptest.NewRequest(http.MethodPost, "/auctions/e13a2778-d1c3-4acf-a771-755ab3cdab4d/participate", bytes.NewBuffer(body))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "e13a2778-d1c3-4acf-a771-755ab3cdab4d")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.Participate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid json")
	})

	t.Run("Participate invalid tenderId UUID", func(t *testing.T) {
		reqBody := participateRequest{
			CompanyID: uuid.MustParse("e13a2778-d1c3-4acf-a771-755ab3cdab4d"),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auctions/not-a-uuid/participate", bytes.NewBuffer(body))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenderId", "not-a-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		h.Participate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid UUID")
	})
}
