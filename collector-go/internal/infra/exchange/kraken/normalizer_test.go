package kraken

import (
	"testing"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

func TestNormalizeTrade(t *testing.T) {
	raw := []byte(`{
		"channel": "trade",
		"type": "update",
		"data": [
			{
				"symbol": "BTC/USD",
				"side": "buy",
				"price": 60000.1,
				"qty": 0.5,
				"ord_type": "market",
				"trade_id": 987,
				"timestamp": "2023-01-01T00:00:00.123456Z"
			}
		]
	}`)
	eventID := "uuid-v7-456"

	trades, err := NormalizeTrade(raw, eventID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	if trade.EventID != eventID {
		t.Errorf("expected EventID %s, got %s", eventID, trade.EventID)
	}
	if trade.Exchange != "kraken" {
		t.Errorf("expected exchange kraken, got %s", trade.Exchange)
	}
	if trade.Side != market.SideBuy {
		t.Errorf("expected side buy, got %d", trade.Side)
	}
	if trade.Price != "60000.10000000" {
		t.Errorf("expected price 60000.10000000, got %s", trade.Price)
	}
}
