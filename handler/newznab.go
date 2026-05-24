package handler

import "net/http"

func ServeNewznab(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")

	switch t {
	case "caps":
		ServeCaps(w, r)
	case "search", "music", "audio":
		ServeSearch(w, r)
	default:
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><error code="202" description="Unknown request type"/>`))
	}
}
