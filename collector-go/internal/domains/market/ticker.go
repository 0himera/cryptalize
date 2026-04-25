package market

// Ticker carries the 24-hour rolling statistics for a trading pair,
// as reported by the exchange.
type Ticker struct {
	// EventID is a time-ordered unique identifier (UUID v7) assigned by the collector.
	EventID string

	// Exchange is the originating exchange slug (e.g. "binance", "kraken").
	Exchange string

	// Pair is the trading pair in canonical slash notation (e.g. "BTC/USDT").
	Pair string

	// Open is the opening price of the 24 h window (decimal string).
	Open string

	// High is the highest price in the 24 h window (decimal string).
	High string

	// Low is the lowest price in the 24 h window (decimal string).
	Low string

	// Close is the last traded price (decimal string).
	Close string

	// Volume is the base-asset volume traded in the 24 h window (decimal string).
	Volume string

	// QuoteVolume is the quote-asset volume traded in the 24 h window (decimal string).
	QuoteVolume string

	// TimestampUs is the snapshot time in Unix microseconds.
	TimestampUs int64
}
