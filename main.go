package main

import (
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

	slog.Info("Audyn started", "port", 5000)
	http.HandleFunc("/", handler.ServeNewznab)

	if err := http.ListenAndServe(":5000", nil); err != nil {
		slog.Error("Audyn stopped", "err", err)
	}
}
