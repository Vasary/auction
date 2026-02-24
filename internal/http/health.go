package http

import (
	"net/http"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "auction-core",
	})
}
