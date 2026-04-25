package kraken

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

// krakenTradeEvent represents the raw JSON structure from Kraken v2 trade stream.
type krakenTradeEvent struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Data    []struct {
		Symbol    string  `json:"symbol"`
		Side      string  `json:"side"`
		Price     float64 `json:"price"`
		Qty       float64 `json:"qty"`
		TradeID   int64   `json:"trade_id"`
		Timestamp string  `json:"timestamp"`
	} `json:"data"`
}

// NormalizeTrade converts a Kraken trade event into a slice of domain Trade models.
// Kraken can batch multiple trades in one message.
func NormalizeTrade(raw []byte, eventID string) ([]market.Trade, error) {
	var ev krakenTradeEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kraken trade: %w", err)
	}

	if ev.Channel != "trade" || ev.Type != "update" {
		return nil, nil // Not a trade update
	}

	trades := make([]market.Trade, 0, len(ev.Data))
	for _, d := range ev.Data {
		side := market.SideUnspecified
		if d.Side == "buy" {
			side = market.SideBuy
		} else if d.Side == "sell" {
			side = market.SideSell
		}

		t, err := time.Parse(time.RFC3339Nano, d.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kraken timestamp %s: %w", d.Timestamp, err)
		}

		trades = append(trades, market.Trade{
			EventID:         eventID,
			ExchangeTradeID: fmt.Sprintf("%d", d.TradeID),
			Exchange:        "kraken",
			Pair:            d.Symbol, // Kraken v2 uses canonical pairs like "BTC/USD"
			Price:           fmt.Sprintf("%.8f", d.Price),
			Quantity:        fmt.Sprintf("%.8f", d.Qty),
			Side:            side,
			TimestampUs:     t.UnixMicro(),
		})
	}

	return trades, nil
}
