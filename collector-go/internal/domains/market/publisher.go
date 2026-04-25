package market

import "context"

// Event is the union type that can be published to the streaming layer.
// Exactly one field must be non-nil per instance.
type Event struct {
	Trade           *Trade
	OrderBookUpdate *OrderBookUpdate
	Ticker          *Ticker
}

// Publisher is the contract for writing normalised market events to the
// streaming layer (Kafka / Redpanda).
//
// Implementations live in internal/infra/kafka/ and are responsible for
// serialising the Event to Protobuf before writing to the wire.
type Publisher interface {
	// Publish writes a single market event to the configured topic.
	// It is safe to call Publish concurrently from multiple goroutines.
	Publish(ctx context.Context, event Event) error

	// Close flushes any buffered messages and releases resources.
	Close() error
}
