package http

import (
	"net/http"

	"auction-core/internal/metrics"
	"auction-core/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WS(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		metrics.WSConnectTotal.WithLabelValues("bad_request").Inc()
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		metrics.WSConnectTotal.WithLabelValues("bad_request").Inc()
		return
	}

	session, exists := h.Manager.Get(parsedTenderID)
	if !exists {
		metrics.WSConnectTotal.WithLabelValues("not_found").Inc()
		h.Error(w, http.StatusNotFound, "auction not active", nil)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		metrics.WSConnectTotal.WithLabelValues("upgrade_error").Inc()
		h.Logger.Error("failed to upgrade to websocket", zap.Error(err))
		return
	}
	metrics.WSConnectTotal.WithLabelValues("ok").Inc()
	metrics.WSConnectionsActive.Inc()

	client := ws.NewClient(
		conn,
		session,
		h.ParticipantRepo,
		parsedTenderID,
		h.Logger,
	)

	go func() {
		defer metrics.WSConnectionsActive.Dec()
		client.Run()
	}()
}
