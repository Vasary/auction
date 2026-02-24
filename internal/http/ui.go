package http

import (
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func (h *Handler) UI(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("ui/dist", r.URL.Path)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.ServeFile(w, r, "ui/dist/index.html")
			return
		}
		h.Logger.Error("failed to open ui file", zap.Error(err), zap.String("path", path))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		h.Logger.Error("failed to stat ui file", zap.Error(err), zap.String("path", path))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		http.ServeFile(w, r, "ui/dist/index.html")
		return
	}

	http.ServeFile(w, r, path)
}

func (h *Handler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/assets/", http.FileServer(http.Dir("ui/dist/assets"))).ServeHTTP(w, r)
}
