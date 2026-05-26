package main

import (
	"audyn/config"
	"audyn/handler"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if cfg.DeezerARL == "" {
		slog.Warn("DEEZER_ARL not set, downloads will fail")
	} else {
		handler.WriteStreamripConfig(cfg.DeezerARL, cfg)
	}

	slog.Info("Audyn started", "port", cfg.Port)

	sabnzbd := &handler.SabnzbdHandler{Config: cfg}
	newznab := &handler.NewznabHandler{Config: cfg}

	http.HandleFunc("/download/api", sabnzbd.Serve)
	http.HandleFunc("/download/config/categories/", sabnzbd.Serve)
	http.HandleFunc("/download/", handler.ServeNZB)
	http.HandleFunc("/", newznab.Serve)

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		slog.Error("Audyn stopped", "err", err)
	}
}
