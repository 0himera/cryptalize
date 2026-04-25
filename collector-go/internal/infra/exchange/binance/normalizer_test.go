package binance

import (
	"testing"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

func TestNormalizeTrade(t *testing.T) {
	// Clean JSON, no extra fields, explicit types
	raw := []byte(`{"E":1672531200000,"s":"BTCUSDT","t":12345,"p":"60000.00","q":"0.1","m":true}`)
	eventID := "uuid-v7-123"

	trade, err := NormalizeTrade(raw, eventID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trade.EventID != eventID {
		t.Errorf("expected EventID %s, got %s", eventID, trade.EventID)
	}
	if trade.Exchange != "binance" {
		t.Errorf("expected exchange binance, got %s", trade.Exchange)
	}
	if trade.Side != market.SideSell {
		t.Errorf("expected side sell (m:true), got %d", trade.Side)
	}
}
