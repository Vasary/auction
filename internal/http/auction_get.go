package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"auction-core/internal/auction"
)

type getAuctionResponse struct {
	TenderID     uuid.UUID      `json:"tenderId"`
	Status       auction.Status `json:"status"`
	StartPrice   int64          `json:"startPrice"`
	Step         int64          `json:"step"`
	CurrentPrice int64          `json:"currentPrice"`
	WinnerID     *uuid.UUID     `json:"winnerId,omitempty"`
	WinnerBidID  *int64         `json:"winnerBidId,omitempty"`
	StartAt      time.Time      `json:"startAt"`
	EndAt        time.Time      `json:"endAt"`
	CreatedBy    uuid.UUID      `json:"createdBy"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Participants []uuid.UUID    `json:"participants"`
}

func (h *Handler) GetAuction(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		return
	}

	a, err := h.AuctionRepo.GetByID(r.Context(), parsedTenderID)
	if err != nil {
		if errors.Is(err, auction.ErrNotFound) {
			h.Error(w, http.StatusNotFound, "auction not found", err)
			return
		}
		h.Error(w, http.StatusInternalServerError, "internal error", err)
		return
	}

	if a == nil {
		h.Error(w, http.StatusNotFound, "auction not found", nil)
		return
	}

	participants, err := h.ParticipantRepo.ListParticipants(r.Context(), parsedTenderID)
	if err != nil {
		h.Logger.Warn("failed to list participants", zap.Error(err), zap.String("tenderId", tenderID))
		participants = []uuid.UUID{} // non-critical, continue with empty list
	}

	resp := getAuctionResponse{
		TenderID:     a.TenderID,
		Status:       a.Status,
		StartPrice:   a.StartPrice,
		Step:         a.Step,
		CurrentPrice: a.CurrentPrice,
		WinnerID:     a.WinnerID,
		WinnerBidID:  a.WinnerBidID,
		StartAt:      a.StartAt,
		EndAt:        a.EndAt,
		CreatedBy:    a.CreatedBy,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		Participants: participants,
	}

	h.JSON(w, http.StatusOK, resp)
}
