package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"nhooyr.io/websocket"
)

const binanceBaseURL = "wss://stream.binance.com:9443/ws"

type Collector struct {
	publisher market.Publisher
	idGen     domains.IDGenerator
	
	mu            sync.RWMutex
	subscriptions map[string]struct{}
	
	conn *websocket.Conn
}

func NewCollector(pub market.Publisher, idGen domains.IDGenerator) *Collector {
	return &Collector{
		publisher:     pub,
		idGen:         idGen,
		subscriptions: make(map[string]struct{}),
	}
}

func (c *Collector) Subscribe(ctx context.Context, pair string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Binance symbols are uppercase and no slash: BTC/USDT -> BTCUSDT
	// This is a simplification; a real implementation would use a mapper.
	symbol := formatSymbol(pair)
	c.subscriptions[symbol] = struct{}{}
	
	if c.conn != nil {
		return c.sendSubscription(ctx, symbol, "SUBSCRIBE")
	}
	return nil
}

func (c *Collector) Unsubscribe(ctx context.Context, pair string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	symbol := formatSymbol(pair)
	delete(c.subscriptions, symbol)
	
	if c.conn != nil {
		return c.sendSubscription(ctx, symbol, "UNSUBSCRIBE")
	}
	return nil
}

func (c *Collector) Run(ctx context.Context) error {
	for {
		err := c.runOnce(ctx)
		if err != nil {
			log.Printf("Binance collector error: %v. Reconnecting...", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Exponential backoff could be added here
			}
		}
	}
}

func (c *Collector) runOnce(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, binanceBaseURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	
	c.mu.Lock()
	c.conn = conn
	// Re-subscribe to existing pairs
	for symbol := range c.subscriptions {
		if err := c.sendSubscription(ctx, symbol, "SUBSCRIBE"); err != nil {
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
		trade, err := NormalizeTrade(message, eventID)
		if err != nil {
			// Some messages might not be trades (e.g., subscription confirmation)
			continue
		}

		err = c.publisher.Publish(ctx, market.Event{Trade: &trade})
		if err != nil {
			log.Printf("Failed to publish binance trade: %v", err)
		}
	}
}

func (c *Collector) sendSubscription(ctx context.Context, symbol string, method string) error {
	payload := map[string]interface{}{
		"method": method,
		"params": []string{fmt.Sprintf("%s@trade", symbol)},
		"id":     1,
	}
	data, _ := json.Marshal(payload)
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func formatSymbol(pair string) string {
	// Simple mapping: BTC/USDT -> btcusdt (Binance expects lowercase in stream names)
	// In production, we'd need a robust exchange-specific mapper.
	var result string
	for _, r := range pair {
		if r != '/' {
			result += string(r)
		}
	}
	// Binance stream names are lowercase
	return lowercase(result)
}

func lowercase(s string) string {
	res := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			res += string(r + ('a' - 'A'))
		} else {
			res += string(r)
		}
	}
	return res
}
