package handler

import (
	"audyn/config"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

type CapsHandler struct {
	Config config.Config
}

var (
	capsOnce  sync.Once
	capsBytes []byte
	capsErr   error
)

func (h *CapsHandler) Serve(w http.ResponseWriter, r *http.Request) {
	slog.Info("Caps requested")
	capsOnce.Do(func() {
		capsBytes, capsErr = os.ReadFile("config/caps.xml")
	})
	if capsErr != nil {
		slog.Error("caps.xml not found", "err", capsErr)
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><error code="300" description="caps.xml not found"/>`)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(capsBytes)
}
