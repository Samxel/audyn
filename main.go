package main

import (
	"audyn/config"
	"audyn/handler"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if cfg.DeezerARL == "" {
		slog.Warn("DEEZER_ARL not set, downloads will fail")
	}

	slog.Info("Audyn started", "port", cfg.Port)

	sabnzbd := &handler.SabnzbdHandler{Config: cfg}
	newznab := &handler.NewznabHandler{Config: cfg}

	http.HandleFunc("/download/api", sabnzbd.Serve)
	http.HandleFunc("/download/config/categories/", sabnzbd.Serve)
	http.HandleFunc("/download/", handler.ServeNZB)
	http.HandleFunc("/", newznab.Serve)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Audyn stopped", "err", err)
	}
}
