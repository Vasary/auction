package http

import (
	"net/http"
	"time"

	"auction-core/internal/auction"

	"github.com/google/uuid"
)

type auctionListItem struct {
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
}

func (h *Handler) ListAuctions(w http.ResponseWriter, r *http.Request) {
	auctions, err := h.AuctionRepo.List(r.Context())
	if err != nil {
		h.Error(w, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]auctionListItem, 0, len(auctions))
	for _, a := range auctions {
		resp = append(resp, auctionListItem{
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
		})
	}

	h.JSON(w, http.StatusOK, resp)
}
