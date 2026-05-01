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

	persisted, now, ok := h.prepareAuction(w, req)
	if !ok {
		return
	}

	if err := h.AuctionRepo.Create(r.Context(), persisted); err != nil {
		h.handleCreateAuctionError(w, err)
		return
	}

	h.createSessionIfNeeded(persisted, now)
	h.JSON(w, http.StatusCreated, newCreateAuctionResponse(persisted))
}

func (h *Handler) prepareAuction(w http.ResponseWriter, req createAuctionRequest) (*auction.PersistedAuction, time.Time, bool) {
	if !validCreateAuctionInput(req) {
		h.Error(w, http.StatusBadRequest, "invalid input", nil)
		return nil, time.Time{}, false
	}

	startAt, ok := h.parseAuctionTime(w, req.StartAt, "startAt")
	if !ok {
		return nil, time.Time{}, false
	}
	endAt, ok := h.parseAuctionTime(w, req.EndAt, "endAt")
	if !ok {
		return nil, time.Time{}, false
	}

	now := time.Now()
	if !h.validAuctionWindow(w, startAt, endAt, now) {
		return nil, time.Time{}, false
	}

	return &auction.PersistedAuction{
		TenderID:     req.TenderID,
		StartPrice:   req.StartPrice,
		Step:         req.Step,
		StartAt:      startAt,
		EndAt:        endAt,
		CreatedBy:    req.CreatedBy,
		Status:       auctionStatusAt(now, startAt, endAt),
		CurrentPrice: req.StartPrice,
		WinnerID:     nil,
	}, now, true
}

func validCreateAuctionInput(req createAuctionRequest) bool {
	return req.TenderID != uuid.Nil &&
		req.StartPrice > 0 &&
		req.Step > 0 &&
		req.CreatedBy != uuid.Nil
}

func (h *Handler) parseAuctionTime(w http.ResponseWriter, value string, field string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		h.Error(w, http.StatusBadRequest, "invalid "+field+" format", err)
		return time.Time{}, false
	}
	return parsed, true
}

func (h *Handler) validAuctionWindow(w http.ResponseWriter, startAt, endAt, now time.Time) bool {
	if !endAt.After(startAt) {
		h.Error(w, http.StatusBadRequest, "endAt must be after startAt", nil)
		return false
	}
	if endAt.Before(now) {
		h.Error(w, http.StatusBadRequest, "endAt cannot be in the past", nil)
		return false
	}
	return true
}

func auctionStatusAt(now, startAt, endAt time.Time) auction.Status {
	if now.After(startAt) && now.Before(endAt) {
		return auction.StatusActive
	}
	if now.After(endAt) {
		return auction.StatusFinished
	}
	return auction.StatusScheduled
}

func (h *Handler) handleCreateAuctionError(w http.ResponseWriter, err error) {
	if errors.Is(err, auction.ErrAlreadyExists) {
		h.Error(w, http.StatusConflict, "auction already exists", err)
		return
	}
	h.Error(w, http.StatusInternalServerError, "failed to save auction", err)
}

func (h *Handler) createSessionIfNeeded(persisted *auction.PersistedAuction, now time.Time) {
	if !shouldCreateSession(persisted.StartAt, persisted.EndAt, now) {
		return
	}

	cfg := auction.Config{
		TenderID:           persisted.TenderID,
		StartPrice:         persisted.StartPrice,
		CurrentPrice:       persisted.StartPrice,
		Step:               persisted.Step,
		StartAt:            persisted.StartAt,
		EndAt:              persisted.EndAt,
		RateLimitPerBidder: 500 * time.Millisecond,
		BroadcastBuffer:    64,
	}
	if _, err := h.Manager.Create(cfg); err != nil {
		h.Logger.Error("failed to create auction session", zap.Error(err), zap.String("tender_id", persisted.TenderID.String()))
	}
}

func shouldCreateSession(startAt, endAt, now time.Time) bool {
	return now.Before(endAt) && (now.After(startAt) || startAt.Sub(now) <= 5*time.Minute)
}

func newCreateAuctionResponse(persisted *auction.PersistedAuction) createAuctionResponse {
	now := time.Now()
	return createAuctionResponse{
		TenderID:     persisted.TenderID,
		Status:       persisted.Status,
		StartPrice:   persisted.StartPrice,
		Step:         persisted.Step,
		CurrentPrice: persisted.CurrentPrice,
		WinnerID:     persisted.WinnerID,
		StartAt:      persisted.StartAt,
		EndAt:        persisted.EndAt,
		CreatedBy:    persisted.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
