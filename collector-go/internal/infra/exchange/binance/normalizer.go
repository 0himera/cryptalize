package binance

import (
	"encoding/json"
	"fmt"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

// tradeEvent represents the raw JSON structure from Binance trade stream.
type tradeEvent struct {
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	TradeID   int64  `json:"t"`
	Price     string `json:"p"`
	Quantity  string `json:"q"`
	IsBuyerMK bool   `json:"m"` // True if the buyer is the market maker (meaning it's a SELL trade)
}

// NormalizeTrade converts a Binance trade event into the domain's Trade model.
func NormalizeTrade(raw []byte, eventID string) (market.Trade, error) {
	var ev tradeEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return market.Trade{}, fmt.Errorf("failed to unmarshal binance trade: %w", err)
	}

	side := market.SideBuy
	if ev.IsBuyerMK {
		side = market.SideSell
	}

	return market.Trade{
		EventID:         eventID,
		ExchangeTradeID: fmt.Sprintf("%d", ev.TradeID),
		Exchange:        "binance",
		Pair:            ev.Symbol, // Note: This is "BTCUSDT", we might need a mapping to "BTC/USDT"
		Price:           ev.Price,
		Quantity:        ev.Quantity,
		Side:            side,
		TimestampUs:     ev.EventTime * 1000, // Binance provides milliseconds
	}, nil
}
