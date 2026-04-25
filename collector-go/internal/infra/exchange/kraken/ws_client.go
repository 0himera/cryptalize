package kraken

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/0himera/cryptalize/collector-go/internal/domains"
	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"github.com/0himera/cryptalize/collector-go/internal/infra/metrics"
	"nhooyr.io/websocket"
	"time"
)

const krakenBaseURL = "wss://ws.kraken.com/v2"

type Collector struct {
	publisher market.Publisher
	idGen     domains.IDGenerator
	snapshots *market.SnapshotStore

	mu            sync.RWMutex
	subscriptions []string

	conn *websocket.Conn
}

func NewCollector(pub market.Publisher, idGen domains.IDGenerator, snapshots *market.SnapshotStore) *Collector {
	return &Collector{
		publisher: pub,
		idGen:     idGen,
		snapshots: snapshots,
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
	defer func() {
		conn.Close(websocket.StatusNormalClosure, "")
		metrics.WSConnected.WithLabelValues("kraken").Set(0)
		c.snapshots.SetConnStatus("kraken", false)
	}()

	c.mu.Lock()
	c.conn = conn
	log.Printf("Connected to Kraken WebSocket")
	metrics.WSConnected.WithLabelValues("kraken").Set(1)
	metrics.WSReconnectsTotal.WithLabelValues("kraken").Inc()
	c.snapshots.SetConnStatus("kraken", true)
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

		metrics.WSMessagesTotal.WithLabelValues("kraken", "raw").Inc()
		eventID := c.idGen.NewID()
		start := time.Now()
		
		trades, err := NormalizeTrade(message, eventID)
		if err == nil && trades != nil {
			metrics.WSMessagesTotal.WithLabelValues("kraken", "trade").Inc()
			for _, t := range trades {
				err = c.publisher.Publish(ctx, market.Event{Trade: &t})
				if err != nil {
					log.Printf("Failed to publish kraken trade: %v", err)
				}
				c.snapshots.UpdateTrade(&t)
			}
			metrics.WSProcessingDuration.WithLabelValues("kraken", "trade").Observe(time.Since(start).Seconds())
			continue
		}

		// Handle Order Books
		books, errBook := NormalizeOrderBook(message, eventID)
		if errBook == nil && books != nil {
			metrics.WSMessagesTotal.WithLabelValues("kraken", "depth").Inc()
			for _, b := range books {
				err = c.publisher.Publish(ctx, market.Event{OrderBookUpdate: &b})
				if err != nil {
					log.Printf("Failed to publish kraken book: %v", err)
				}
				c.snapshots.UpdateOrderBook(&b)
			}
			metrics.WSProcessingDuration.WithLabelValues("kraken", "depth").Observe(time.Since(start).Seconds())
			continue
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
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}

	payloadBook := map[string]interface{}{
		"method": method,
		"params": map[string]interface{}{
			"channel": "book",
			"depth":   10,
			"symbol":  symbols,
		},
	}
	dataBook, _ := json.Marshal(payloadBook)
	return c.conn.Write(ctx, websocket.MessageText, dataBook)
}
