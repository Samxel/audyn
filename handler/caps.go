package handler

import (
	"log/slog"
	"net/http"
	"os"
)

func ServeCaps(w http.ResponseWriter, r *http.Request) {
	slog.Info("Caps requested")
	dat, err := os.ReadFile("config/caps.xml")
	if err != nil {
		slog.Error("caps.xml not found", "err", err)
		http.Error(w, "caps.xml not found", 500)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	w.Write(dat)
}
