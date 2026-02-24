package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"auction-core/internal/auction"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type createAuctionRequest struct {
	TenderID   uuid.UUID `json:"tenderId"`
	StartPrice int64     `json:"startPrice"`
	Step       int64     `json:"step"`
	StartAt    string    `json:"startAt"`
	EndAt      string    `json:"endAt"`
	CreatedBy  uuid.UUID `json:"createdBy"`
}

type createAuctionResponse struct {
	TenderID     uuid.UUID      `json:"tenderId"`
	Status       auction.Status `json:"status"`
	StartPrice   int64          `json:"startPrice"`
	Step         int64          `json:"step"`
	CurrentPrice int64          `json:"currentPrice"`
	WinnerID     *uuid.UUID     `json:"winnerId,omitempty"`
	StartAt      time.Time      `json:"startAt"`
	EndAt        time.Time      `json:"endAt"`
	CreatedBy    uuid.UUID      `json:"createdBy"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

func (h *Handler) CreateAuction(w http.ResponseWriter, r *http.Request) {
	var req createAuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, http.StatusBadRequest, "invalid json", err)
		return
	}

	if req.TenderID == uuid.Nil || req.StartPrice <= 0 || req.Step <= 0 || req.CreatedBy == uuid.Nil {
		h.Error(w, http.StatusBadRequest, "invalid input", nil)
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		h.Error(w, http.StatusBadRequest, "invalid startAt format", err)
		return
	}

	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		h.Error(w, http.StatusBadRequest, "invalid endAt format", err)
		return
	}

	if !endAt.After(startAt) {
		h.Error(w, http.StatusBadRequest, "endAt must be after startAt", nil)
		return
	}

	now := time.Now()
	if endAt.Before(now) {
		h.Error(w, http.StatusBadRequest, "endAt cannot be in the past", nil)
		return
	}

	status := auction.StatusScheduled
	if now.After(startAt) && now.Before(endAt) {
		status = auction.StatusActive
	}
	if now.After(endAt) {
		status = auction.StatusFinished
	}

	persisted := &auction.PersistedAuction{
		TenderID:     req.TenderID,
		StartPrice:   req.StartPrice,
		Step:         req.Step,
		StartAt:      startAt,
		EndAt:        endAt,
		CreatedBy:    req.CreatedBy,
		Status:       status,
		CurrentPrice: req.StartPrice,
		WinnerID:     nil,
	}

	if err := h.AuctionRepo.Create(r.Context(), persisted); err != nil {
		if errors.Is(err, auction.ErrAlreadyExists) {
			h.Error(w, http.StatusConflict, "auction already exists", err)
			return
		}
		h.Error(w, http.StatusInternalServerError, "failed to save auction", err)
		return
	}

	if now.Before(endAt) && (now.After(startAt) || startAt.Sub(now) <= 5*time.Minute) {
		cfg := auction.Config{
			TenderID:           req.TenderID,
			StartPrice:         req.StartPrice,
			CurrentPrice:       req.StartPrice,
			Step:               req.Step,
			StartAt:            startAt,
			EndAt:              endAt,
			RateLimitPerBidder: 500 * time.Millisecond,
			BroadcastBuffer:    64,
		}
		if _, err := h.Manager.Create(cfg); err != nil {
			h.Logger.Error("failed to create auction session", zap.Error(err), zap.String("tender_id", req.TenderID.String()))
		}
	}

	resp := createAuctionResponse{
		TenderID:     persisted.TenderID,
		Status:       persisted.Status,
		StartPrice:   persisted.StartPrice,
		Step:         persisted.Step,
		CurrentPrice: persisted.CurrentPrice,
		WinnerID:     persisted.WinnerID,
		StartAt:      persisted.StartAt,
		EndAt:        persisted.EndAt,
		CreatedBy:    persisted.CreatedBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.JSON(w, http.StatusCreated, resp)
}
