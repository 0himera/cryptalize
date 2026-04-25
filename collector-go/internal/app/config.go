package app

import (
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	
	BinancePairs []string
	KrakenPairs  []string
}

func LoadConfig() Config {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:19092"
	}
	
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "market.events"
	}
	
	binancePairs := os.Getenv("BINANCE_PAIRS")
	if binancePairs == "" {
		binancePairs = "BTC/USDT,ETH/USDT"
	}
	
	krakenPairs := os.Getenv("KRAKEN_PAIRS")
	if krakenPairs == "" {
		krakenPairs = "BTC/USD,ETH/USD"
	}

	return Config{
		KafkaBrokers: strings.Split(brokers, ","),
		KafkaTopic:   topic,
		BinancePairs: strings.Split(binancePairs, ","),
		KrakenPairs:  strings.Split(krakenPairs, ","),
	}
}
