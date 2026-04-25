package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/0himera/cryptalize/collector-go/internal/app"
	"github.com/0himera/cryptalize/collector-go/internal/infra"
	"github.com/0himera/cryptalize/collector-go/internal/infra/exchange/binance"
	"github.com/0himera/cryptalize/collector-go/internal/infra/exchange/kraken"
	"github.com/0himera/cryptalize/collector-go/internal/infra/kafka"
)

func main() {
	cfg := app.LoadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Infrastructure
	idGen := infra.UUIDGenerator{}
	
	publisher, err := kafka.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka publisher: %v", err)
	}
	defer publisher.Close()

	// 2. Initialize Collectors
	binanceCollector := binance.NewCollector(publisher, idGen)
	krakenCollector := kraken.NewCollector(publisher, idGen)

	// 3. Subscribe to Pairs
	for _, pair := range cfg.BinancePairs {
		if err := binanceCollector.Subscribe(ctx, pair); err != nil {
			log.Printf("Failed to subscribe Binance to %s: %v", pair, err)
		}
	}
	for _, pair := range cfg.KrakenPairs {
		if err := krakenCollector.Subscribe(ctx, pair); err != nil {
			log.Printf("Failed to subscribe Kraken to %s: %v", pair, err)
		}
	}

	// 4. Start Collectors
	go func() {
		if err := binanceCollector.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("Binance collector exited with error: %v", err)
		}
	}()
	go func() {
		if err := krakenCollector.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("Kraken collector exited with error: %v", err)
		}
	}()

	// 5. Start HTTP Server (for health checks / metrics)
	srv := http.Server{
		Addr:    ":8000",
		Handler: buildHTTPHandler(),
	}

	go func() {
		log.Printf("Running HTTP server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 6. Wait for Shutdown Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel() // Stop collectors
	srv.Shutdown(context.Background())
	log.Println("Shutdown complete.")
}
