package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

type SnapshotHandler struct {
	snapshots *market.SnapshotStore
}

func CreateSnapshotHandler(snapshots *market.SnapshotStore) *SnapshotHandler {
	return &SnapshotHandler{snapshots: snapshots}
}

func (h *SnapshotHandler) GetTickers(w http.ResponseWriter, r *http.Request) {
	tickers := h.snapshots.GetTickers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickers)
}

func (h *SnapshotHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.snapshots.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
