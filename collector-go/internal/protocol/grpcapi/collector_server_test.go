package grpcapi

import (
	"context"
	"testing"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	collectorv1 "github.com/0himera/cryptalize/collector-go/internal/proto/collector/v1"
)

// mockCollector implements market.Collector for testing.
type mockCollector struct {
	subscribed bool
	pair       string
}

func (m *mockCollector) Subscribe(ctx context.Context, pair string) error {
	m.subscribed = true
	m.pair = pair
	return nil
}

func (m *mockCollector) Unsubscribe(ctx context.Context, pair string) error {
	m.subscribed = false
	m.pair = ""
	return nil
}

func (m *mockCollector) Run(ctx context.Context) error {
	return nil
}

func TestCollectorServer_Subscribe(t *testing.T) {
	mockBinance := &mockCollector{}
	collectors := map[string]market.Collector{
		"binance": mockBinance,
	}
	server := NewCollectorServer(collectors)

	// 1. Success case
	resp, err := server.Subscribe(context.Background(), &collectorv1.SubscribeRequest{
		Exchange: "binance",
		Pair:     "BTC/USDT",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, got false: %s", resp.Message)
	}
	if !mockBinance.subscribed || mockBinance.pair != "BTC/USDT" {
		t.Errorf("mock collector was not updated correctly")
	}

	// 2. Exchange not found
	resp, err = server.Subscribe(context.Background(), &collectorv1.SubscribeRequest{
		Exchange: "unknown",
		Pair:     "BTC/USDT",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Errorf("expected failure for unknown exchange")
	}
}
