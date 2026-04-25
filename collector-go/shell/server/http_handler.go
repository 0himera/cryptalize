package main

import (
	"net/http"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"github.com/0himera/cryptalize/collector-go/internal/infra"
	"github.com/0himera/cryptalize/collector-go/internal/protocol/httpapi"
)

func buildHTTPHandler(snapshots *market.SnapshotStore) http.Handler {
	idGen:= infra.UUIDGenerator{}

	clock := infra.SystemClock{}
	systemCloclService := domains.NewDatetimeService(clock)
	snapshotHandler := httpapi.CreateSnapshotHandler(snapshots)

	mux:= http.NewServeMux()
	mux.Handle("GET /", httpapi.CreateDatetimeHandler(systemCloclService))
	mux.HandleFunc("GET /healthz", httpapi.HealthzHandler)
	mux.HandleFunc("GET /snapshot/tickers", snapshotHandler.GetTickers)
	mux.HandleFunc("GET /snapshot/status", snapshotHandler.GetStatus)

	var handler http.Handler = mux
	handler = httpapi.RequestIDMiddleware(idGen)(handler)

	return handler
}
