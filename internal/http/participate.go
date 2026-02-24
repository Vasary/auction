package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"auction-core/internal/auction"

	"github.com/google/uuid"
)

type participateRequest struct {
	CompanyID uuid.UUID `json:"companyId"`
}

func (h *Handler) Participate(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		return
	}

	var req participateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, http.StatusBadRequest, "invalid json", err)
		return
	}

	if req.CompanyID == uuid.Nil {
		h.Error(w, http.StatusBadRequest, "missing companyId", nil)
		return
	}

	a, err := h.AuctionRepo.GetByID(r.Context(), parsedTenderID)
	if err != nil {
		h.Error(w, http.StatusNotFound, "auction not found", err)
		return
	}

	if a.Status == auction.StatusFinished {
		h.Error(w, http.StatusBadRequest, "auction already finished", nil)
		return
	}

	if err := h.ParticipantRepo.AddParticipant(
		r.Context(),
		parsedTenderID,
		req.CompanyID,
	); err != nil {
		h.Error(w, http.StatusInternalServerError, "failed to register participant", err)
		return
	}

	h.JSON(w, http.StatusCreated, map[string]any{
		"success":    true,
		"tenderId":   parsedTenderID,
		"companyId":  req.CompanyID,
		"registered": true,
	})
}
