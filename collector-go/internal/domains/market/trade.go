package market

// Trade represents a single executed trade on an exchange,
// normalised from the exchange's raw WebSocket format into
// the canonical domain representation.
//
// Prices and quantities are held as strings to preserve full
// decimal precision — identical to the Protobuf wire format.
type Trade struct {
	// EventID is a time-ordered unique identifier (UUID v7) generated
	// by the collector at normalisation time.
	EventID string

	// ExchangeTradeID is the originating exchange's own trade identifier.
	// May be empty if the venue does not provide one.
	ExchangeTradeID string

	// Exchange is the originating exchange slug (e.g. "binance", "kraken").
	Exchange string

	// Pair is the trading pair in canonical slash notation (e.g. "BTC/USDT").
	Pair string

	// Price is the executed price as a decimal string (e.g. "67432.15").
	Price string

	// Quantity is the executed quantity as a decimal string (e.g. "0.00423").
	Quantity string

	// Side is the aggressor side of the trade.
	Side BuySell

	// TimestampUs is the exchange-reported execution time in Unix microseconds.
	TimestampUs int64
}

// BuySell represents the aggressor side of a trade.
type BuySell uint8

const (
	SideUnspecified BuySell = iota
	SideBuy
	SideSell
)
