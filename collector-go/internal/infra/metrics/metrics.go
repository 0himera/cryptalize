package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WSMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "collector_ws_messages_total",
			Help: "Total number of WebSocket messages received from exchanges",
		},
		[]string{"exchange", "stream_type"},
	)

	WSReconnectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "collector_ws_reconnects_total",
			Help: "Total number of WebSocket reconnections",
		},
		[]string{"exchange"},
	)

	WSConnected = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_ws_connected",
			Help: "WebSocket connection status (1=connected, 0=disconnected)",
		},
		[]string{"exchange"},
	)

	WSProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "collector_ws_processing_duration_seconds",
			Help:    "Time taken to normalize and prepare WS message for publishing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"exchange", "stream_type"},
	)

	KafkaPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "collector_kafka_publish_duration_seconds",
			Help:    "Time taken to publish message to Kafka/Redpanda",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic"},
	)

	KafkaPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "collector_kafka_publish_total",
			Help: "Total number of messages published to Kafka",
		},
		[]string{"topic", "status"},
	)

	EventsPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_events_per_second",
			Help: "Events processed per second (calculated window)",
		},
		[]string{"exchange"},
	)
)
