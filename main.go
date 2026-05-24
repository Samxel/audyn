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

	slog.Info("Audyn started", "port", cfg.Port)
	newznab := &handler.NewznabHandler{Config: cfg}
	http.HandleFunc("/", newznab.Serve)

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		slog.Error("Audyn stopped", "err", err)
	}
}
