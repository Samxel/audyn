package handler

import (
	"audyn/config"
	"net/http"
)

type NewznabHandler struct {
	Config config.Config
}

func (h *NewznabHandler) Serve(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")

	switch t {
	case "caps":
		caps := &CapsHandler{Config: h.Config}
		caps.Serve(w, r)
	case "search", "music", "audio":
		search := &SearchHandler{Config: h.Config}
		search.Serve(w, r)
	default:
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><error code="202" description="Unknown request type"/>`))
	}
}
