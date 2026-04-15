package handlers

import (
	"context"
	"encoding/json"
	"net/http"
)

var readinessProbe func(context.Context) error

func InitReadiness(probe func(context.Context) error) {
	readinessProbe = probe
}

func HandleHealth(w http.ResponseWriter, _ *http.Request) {
	response := struct {
		Status string `json:"status"`
		Text   string `json:"text"`
	}{
		Status: "success",
		Text:   "healthy",
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	if readinessProbe != nil {
		if err := readinessProbe(r.Context()); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "service not ready")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"text":   "ready",
	})
}
