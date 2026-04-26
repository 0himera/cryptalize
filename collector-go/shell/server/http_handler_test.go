package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

func TestBuildHTTPHandler(t *testing.T) {
	snapshots := market.NewSnapshotStore()
	handler := buildHTTPHandler(snapshots)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected json content type, got %s", contentType)
	}

	requestID := rr.Header().Get("Request-ID")
	if requestID == "" {
		t.Errorf("expected Request-ID header to be set")
	}
}
