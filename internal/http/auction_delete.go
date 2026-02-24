package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"auction-core/internal/auction"
)

func (h *Handler) DeleteAuction(w http.ResponseWriter, r *http.Request) {
	tenderID := chi.URLParam(r, "tenderId")
	if tenderID == "" {
		h.Error(w, http.StatusBadRequest, "missing tenderId", nil)
		return
	}

	parsedTenderID, ok := h.parseUUID(w, tenderID, "tenderId")
	if !ok {
		return
	}

	err := h.Manager.Delete(r.Context(), parsedTenderID)
	if err != nil {
		if errors.Is(err, auction.ErrNotFound) {
			h.Error(w, http.StatusNotFound, "auction not found", err)
			return
		}
		if errors.Is(err, auction.ErrCannotDeleteStarted) {
			h.Error(w, http.StatusConflict, err.Error(), err)
			return
		}
		h.Error(w, http.StatusInternalServerError, "internal error", err)
		return
	}

	h.JSON(w, http.StatusNoContent, nil)
}
