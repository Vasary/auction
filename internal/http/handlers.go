package http

import (
	"encoding/json"
	"net/http"

	"auction-core/internal/auction"
	"auction-core/internal/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Handler struct {
	Manager         *auction.Manager
	ParticipantRepo auction.ParticipantRepository
	AuctionRepo     auction.AuctionsRepository
	BidRepo         auction.BidRepository
	Logger          *zap.Logger
}

func NewHandler(
	manager *auction.Manager,
	participantRepo auction.ParticipantRepository,
	auctionRepo auction.AuctionsRepository,
	bidRepo auction.BidRepository,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		Manager:         manager,
		ParticipantRepo: participantRepo,
		AuctionRepo:     auctionRepo,
		BidRepo:         bidRepo,
		Logger:          logger,
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/ws/{tenderId}", h.WS)

	r.Group(func(gr chi.Router) {
		gr.Use(metrics.Middleware)

		gr.Get("/health", h.Health)
		gr.Mount("/ui", http.HandlerFunc(h.UI))
		gr.Get("/assets/*", h.ServeStatic)

		gr.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics.Collect(h.Manager)
			promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		}))

		gr.Post("/auctions", h.CreateAuction)
		gr.Get("/auctions", h.ListAuctions)
		gr.Get("/auctions/{tenderId}", h.GetAuction)
		gr.Patch("/auctions/{tenderId}", h.UpdateAuction)
		gr.Delete("/auctions/{tenderId}", h.DeleteAuction)
		gr.Post("/auctions/{tenderId}/participate", h.Participate)
		gr.Get("/auctions/{tenderId}/bids", h.ListBids)
	})

	return r
}

func (h *Handler) JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.Logger.Error("failed to encode json response", zap.Error(err))
		}
	}
}

func (h *Handler) Error(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		h.Logger.Error(message, zap.Error(err), zap.Int("status", status))
	}
	h.JSON(w, status, map[string]string{"error": message})
}

func (h *Handler) parseUUID(w http.ResponseWriter, id string, name string) (uuid.UUID, bool) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		h.Error(w, http.StatusBadRequest, "invalid UUID: "+name, err)
		return uuid.Nil, false
	}
	return parsed, true
}
