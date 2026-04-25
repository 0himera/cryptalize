package kafka

import (
	"context"
	"fmt"
	"log"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	"github.com/0himera/cryptalize/collector-go/internal/infra/metrics"
	marketv1 "github.com/0himera/cryptalize/collector-go/internal/proto/market/v1"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
	"time"
)

type Publisher struct {
	client *kgo.Client
	topic  string
}

func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Publisher{
		client: client,
		topic:  topic,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, event market.Event) error {
	log.Printf("Publishing event to topic %s", p.topic)
	var protoEvent marketv1.MarketEvent

	if event.Trade != nil {
		protoEvent.Payload = &marketv1.MarketEvent_Trade{
			Trade: &marketv1.Trade{
				EventId:         event.Trade.EventID,
				ExchangeTradeId: event.Trade.ExchangeTradeID,
				Exchange:        event.Trade.Exchange,
				Pair:            event.Trade.Pair,
				Price:           event.Trade.Price,
				Quantity:        event.Trade.Quantity,
				Side:            marketv1.BuySell(event.Trade.Side),
				TimestampUs:     event.Trade.TimestampUs,
			},
		}
	} else if event.OrderBookUpdate != nil {
		bids := make([]*marketv1.PriceLevel, len(event.OrderBookUpdate.Bids))
		for i, b := range event.OrderBookUpdate.Bids {
			bids[i] = &marketv1.PriceLevel{Price: b.Price, Quantity: b.Quantity}
		}
		asks := make([]*marketv1.PriceLevel, len(event.OrderBookUpdate.Asks))
		for i, a := range event.OrderBookUpdate.Asks {
			asks[i] = &marketv1.PriceLevel{Price: a.Price, Quantity: a.Quantity}
		}

		protoEvent.Payload = &marketv1.MarketEvent_OrderBookUpdate{
			OrderBookUpdate: &marketv1.OrderBookUpdate{
				EventId:     event.OrderBookUpdate.EventID,
				Exchange:    event.OrderBookUpdate.Exchange,
				Pair:        event.OrderBookUpdate.Pair,
				Sequence:    event.OrderBookUpdate.Sequence,
				Bids:        bids,
				Asks:        asks,
				TimestampUs: event.OrderBookUpdate.TimestampUs,
			},
		}
	} else if event.Ticker != nil {
		protoEvent.Payload = &marketv1.MarketEvent_Ticker{
			Ticker: &marketv1.Ticker{
				EventId:     event.Ticker.EventID,
				Exchange:    event.Ticker.Exchange,
				Pair:        event.Ticker.Pair,
				Open:        event.Ticker.Open,
				High:        event.Ticker.High,
				Low:         event.Ticker.Low,
				Close:       event.Ticker.Close,
				Volume:      event.Ticker.Volume,
				QuoteVolume: event.Ticker.QuoteVolume,
				TimestampUs: event.Ticker.TimestampUs,
			},
		}
	} else {
		return fmt.Errorf("empty event payload")
	}

	data, err := proto.Marshal(&protoEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal proto event: %w", err)
	}

	record := &kgo.Record{
		Topic: p.topic,
		Value: data,
	}

	// We use ProduceSync for simplicity in this initial phase.
	// In production, we'd use asynchronous producing with callbacks for performance.
	start := time.Now()
	results := p.client.ProduceSync(ctx, record)
	duration := time.Since(start).Seconds()
	
	metrics.KafkaPublishDuration.WithLabelValues(p.topic).Observe(duration)

	if err := results.FirstErr(); err != nil {
		metrics.KafkaPublishTotal.WithLabelValues(p.topic, "error").Inc()
		return fmt.Errorf("failed to produce kafka record: %w", err)
	}

	metrics.KafkaPublishTotal.WithLabelValues(p.topic, "success").Inc()
	return nil
}

func (p *Publisher) Close() error {
	p.client.Close()
	return nil
}
