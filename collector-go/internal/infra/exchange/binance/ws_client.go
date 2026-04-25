package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"github.com/0himera/cryptalize/collector-go/internal/infra/metrics"
	"nhooyr.io/websocket"
	"time"
)

const binanceBaseURL = "wss://stream.binance.com:9443/stream"

type combinedMessage struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type Collector struct {
	publisher market.Publisher
	idGen     domains.IDGenerator
	snapshots *market.SnapshotStore
	
	mu            sync.RWMutex
	subscriptions map[string]struct{}
	
	conn *websocket.Conn
}

func NewCollector(pub market.Publisher, idGen domains.IDGenerator, snapshots *market.SnapshotStore) *Collector {
	return &Collector{
		publisher:     pub,
		idGen:         idGen,
		snapshots:     snapshots,
		subscriptions: make(map[string]struct{}),
	}
}

func (c *Collector) Subscribe(ctx context.Context, pair string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
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
			}
		}
	}
}

func (c *Collector) runOnce(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, binanceBaseURL, nil)
	if err != nil {
		return err
	}
	defer func() {
		conn.Close(websocket.StatusNormalClosure, "")
		metrics.WSConnected.WithLabelValues("binance").Set(0)
		c.snapshots.SetConnStatus("binance", false)
	}()
	
	c.mu.Lock()
	c.conn = conn
	log.Printf("Connected to Binance WebSocket (Combined Stream)")
	metrics.WSConnected.WithLabelValues("binance").Set(1)
	metrics.WSReconnectsTotal.WithLabelValues("binance").Inc()
	c.snapshots.SetConnStatus("binance", true)
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

		var combined combinedMessage
		if err := json.Unmarshal(message, &combined); err != nil {
			continue
		}
		
		metrics.WSMessagesTotal.WithLabelValues("binance", "raw").Inc()

		eventID := c.idGen.NewID()
		start := time.Now()
		
		// Handle Trades
		trade, errTrade := NormalizeTrade(combined.Data, eventID)
		if errTrade == nil {
			metrics.WSMessagesTotal.WithLabelValues("binance", "trade").Inc()
			log.Printf("Publishing binance trade: %s", trade.Pair)
			c.publisher.Publish(ctx, market.Event{Trade: &trade})
			c.snapshots.UpdateTrade(&trade)
			metrics.WSProcessingDuration.WithLabelValues("binance", "trade").Observe(time.Since(start).Seconds())
			continue
		}

		// Handle Tickers
		ticker, errTicker := NormalizeTicker(combined.Data, eventID)
		if errTicker == nil {
			metrics.WSMessagesTotal.WithLabelValues("binance", "ticker").Inc()
			log.Printf("Publishing binance ticker: %s", ticker.Pair)
			c.publisher.Publish(ctx, market.Event{Ticker: &ticker})
			c.snapshots.UpdateTicker(&ticker)
			metrics.WSProcessingDuration.WithLabelValues("binance", "ticker").Observe(time.Since(start).Seconds())
			continue
		}
		
		// If we reached here, both failed. Let's see why if it's a known stream.
		if combined.Stream != "" {
			var m map[string]interface{}
			json.Unmarshal(combined.Data, &m)
			log.Printf("Normalization failed for stream %s. TradeErr: %v, TickerErr: %v. Raw types: E=%T, s=%T, p=%T", 
				combined.Stream, errTrade, errTicker, m["E"], m["s"], m["p"])
		}
		
		// Log unknown messages to help debug
		// log.Printf("Unknown binance message on stream %s", combined.Stream)
	}
}

func (c *Collector) sendSubscription(ctx context.Context, symbol string, method string) error {
	payload := map[string]interface{}{
		"method": method,
		"params": []string{
			fmt.Sprintf("%s@trade", symbol),
			fmt.Sprintf("%s@ticker", symbol),
		},
		"id": 1,
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
