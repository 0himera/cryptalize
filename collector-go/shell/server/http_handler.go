package main

import (
	"net/http"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/infra"
	"github.com/0himera/cryptalize/collector-go/internal/protocol/httpapi"
)

func buildHTTPHandler() http.Handler {
	idGen:= infra.UUIDGenerator{}

	clock := infra.SystemClock{}
	systemCloclService := domains.NewDatetimeService(clock)

	mux:= http.NewServeMux()
	mux.Handle("GET /", httpapi.CreateDatetimeHandler(systemCloclService))

	var handler http.Handler = mux
	handler = httpapi.RequestIDMiddleware(idGen)(handler)

	return handler
}
