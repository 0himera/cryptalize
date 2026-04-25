package market

// PriceLevel represents a single entry in an order book at a given price.
// A Quantity of "0" signals that this level should be removed.
type PriceLevel struct {
	// Price of this level (decimal string).
	Price string

	// Quantity available at this level (decimal string).
	// "0" means the level is deleted.
	Quantity string
}

// OrderBookUpdate carries an incremental delta for a single trading pair's
// order book. Consumers apply deltas sequentially to reconstruct a full book.
type OrderBookUpdate struct {
	// EventID is a time-ordered unique identifier (UUID v7) assigned by the collector.
	EventID string

	// Exchange is the originating exchange slug (e.g. "binance", "kraken").
	Exchange string

	// Pair is the trading pair in canonical slash notation (e.g. "BTC/USDT").
	Pair string

	// Sequence is the exchange-provided ordering number.
	// Zero if the exchange does not provide one.
	Sequence int64

	// Bids contains updated bid (buy) levels.
	Bids []PriceLevel

	// Asks contains updated ask (sell) levels.
	Asks []PriceLevel

	// TimestampUs is the exchange-reported update time in Unix microseconds.
	TimestampUs int64
}
