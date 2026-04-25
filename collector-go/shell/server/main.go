package main

import (
	"net/http"
	"log"
)

func main() {
	srv := http.Server{
		Addr: ":8000",
		Handler: buildHTTPHandler(),
	}

	log.Printf("Running server on %s", srv.Addr)
	srv.ListenAndServe()
}
