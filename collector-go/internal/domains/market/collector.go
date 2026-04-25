package market

import "context"

// Collector is the contract that every exchange adapter must satisfy.
//
// It abstracts over the transport (WebSocket) and exchange-specific
// subscription logic. Implementations live in internal/infra/exchange/.
type Collector interface {
	// Subscribe instructs the collector to begin streaming market data
	// for the given canonical pair (e.g. "BTC/USDT").
	// Returns an error if the pair is invalid or the subscription fails.
	Subscribe(ctx context.Context, pair string) error

	// Unsubscribe stops streaming market data for the given pair.
	// Returns an error if the pair is not currently subscribed.
	Unsubscribe(ctx context.Context, pair string) error

	// Run starts the collector's event loop.  It blocks until ctx is
	// cancelled or a fatal error occurs.
	Run(ctx context.Context) error
}
