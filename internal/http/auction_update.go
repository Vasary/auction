package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"auction-core/internal/auction"

	"github.com/go-chi/chi/v5"
)

type updateAuctionRequest struct {
	StartPrice *int64  `json:"startPrice"`
	Step       *int64  `json:"step"`
	StartAt    *string `json:"startAt"`
	EndAt      *string `json:"endAt"`
}

func (h *Handler) UpdateAuction(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		return
	}

	var req updateAuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, http.StatusBadRequest, "invalid json", err)
		return
	}

	err := h.Manager.Update(r.Context(), parsedTenderID, func(a *auction.PersistedAuction) {
		if req.StartPrice != nil {
			a.StartPrice = *req.StartPrice
			a.CurrentPrice = *req.StartPrice
		}
		if req.Step != nil {
			a.Step = *req.Step
		}
		if req.StartAt != nil {
			if t, err := time.Parse(time.RFC3339, *req.StartAt); err == nil {
				a.StartAt = t
			}
		}
		if req.EndAt != nil {
			if t, err := time.Parse(time.RFC3339, *req.EndAt); err == nil {
				a.EndAt = t
			}
		}
	})

	if err != nil {
		if errors.Is(err, auction.ErrNotFound) {
			h.Error(w, http.StatusNotFound, "auction not found", err)
			return
		}
		if errors.Is(err, auction.ErrCannotUpdateStarted) {
			h.Error(w, http.StatusForbidden, "cannot update started or already loaded auction", err)
			return
		}
		if errors.Is(err, auction.ErrCannotUpdateFinished) {
			h.Error(w, http.StatusForbidden, "cannot update finished auction", err)
			return
		}
		if errors.Is(err, auction.ErrInvalidConfig) {
			h.Error(w, http.StatusBadRequest, "invalid auction configuration (check dates)", err)
			return
		}
		h.Error(w, http.StatusInternalServerError, "failed to update auction", err)
		return
	}

	h.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
