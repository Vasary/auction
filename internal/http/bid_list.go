package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type bidResponse struct {
	ID        int64     `json:"id"`
	CompanyID uuid.UUID `json:"companyId"`
	PersonID  uuid.UUID `json:"personId"`
	BidAmount int64     `json:"bidAmount"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handler) ListBids(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		return
	}

	bids, err := h.BidRepo.ListBids(r.Context(), parsedTenderID)
	if err != nil {
		h.Error(w, http.StatusInternalServerError, "failed to fetch bids", err)
		return
	}

	resp := make([]bidResponse, 0, len(bids))

	for _, b := range bids {
		resp = append(resp, bidResponse{
			ID:        b.ID,
			CompanyID: b.CompanyID,
			PersonID:  b.PersonID,
			BidAmount: b.BidAmount,
			CreatedAt: b.CreatedAt,
		})
	}

	h.JSON(w, http.StatusOK, resp)
}
