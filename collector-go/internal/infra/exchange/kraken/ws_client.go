package kraken

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"nhooyr.io/websocket"
)

const krakenBaseURL = "wss://ws.kraken.com/v2"

type Collector struct {
	publisher market.Publisher
	idGen     domains.IDGenerator

	mu            sync.RWMutex
	subscriptions []string

	conn *websocket.Conn
}

func NewCollector(pub market.Publisher, idGen domains.IDGenerator) *Collector {
	return &Collector{
		publisher: pub,
		idGen:     idGen,
	}
}

func (c *Collector) Subscribe(ctx context.Context, pair string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriptions = append(c.subscriptions, pair)

	if c.conn != nil {
		return c.sendSubscription(ctx, []string{pair}, "subscribe")
	}
	return nil
}

func (c *Collector) Unsubscribe(ctx context.Context, pair string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, s := range c.subscriptions {
		if s == pair {
			c.subscriptions = append(c.subscriptions[:i], c.subscriptions[i+1:]...)
			break
		}
	}

	if c.conn != nil {
		return c.sendSubscription(ctx, []string{pair}, "unsubscribe")
	}
	return nil
}

func (c *Collector) Run(ctx context.Context) error {
	for {
		err := c.runOnce(ctx)
		if err != nil {
			log.Printf("Kraken collector error: %v. Reconnecting...", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Backoff logic could go here
			}
		}
	}
}

func (c *Collector) runOnce(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, krakenBaseURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	c.mu.Lock()
	c.conn = conn
	if len(c.subscriptions) > 0 {
		if err := c.sendSubscription(ctx, c.subscriptions, "subscribe"); err != nil {
			c.mu.Unlock()
			return err
		}
	}
	c.mu.Unlock()

	for {
		_, message, err := conn.Read(ctx)
		if err != nil {
			return err
		}

		eventID := c.idGen.NewID()
		trades, err := NormalizeTrade(message, eventID)
		if err != nil {
			// Skip heartbeat or other messages
			continue
		}

		for _, t := range trades {
			err = c.publisher.Publish(ctx, market.Event{Trade: &t})
			if err != nil {
				log.Printf("Failed to publish kraken trade: %v", err)
			}
		}
	}
}

func (c *Collector) sendSubscription(ctx context.Context, symbols []string, method string) error {
	payload := map[string]interface{}{
		"method": method,
		"params": map[string]interface{}{
			"channel": "trade",
			"symbol":  symbols,
		},
	}
	data, _ := json.Marshal(payload)
	return c.conn.Write(ctx, websocket.MessageText, data)
}
