package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestContextMiddlewareSetsIDs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := RequestContext(logger)

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatalf("request id missing from context")
		}
		if CorrelationIDFromContext(r.Context()) == "" {
			t.Fatalf("correlation id missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID response header")
	}
	if rr.Header().Get("X-Correlation-ID") == "" {
		t.Fatalf("expected X-Correlation-ID response header")
	}
}
