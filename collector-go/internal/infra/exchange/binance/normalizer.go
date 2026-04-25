package binance

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
)

// tradeEvent represents the raw JSON structure from Binance trade stream.
type tradeEvent struct {
	EventType       string      `json:"e"`
	EventTime       json.Number `json:"E"`
	Symbol          string      `json:"s"`
	TradeID         json.Number `json:"t"`
	Price           string      `json:"p"`
	Quantity        string      `json:"q"`
	BuyerOrderID    json.Number `json:"b"`
	SellerOrderID   json.Number `json:"a"`
	TradeTime       json.Number `json:"T"`
	IsBuyerMK       bool        `json:"m"`
	Ignore          bool        `json:"M"`
}

// depthEvent represents the raw JSON structure from Binance partial depth stream.
type depthEvent struct {
	LastUpdateID json.Number `json:"lastUpdateId"`
	Bids         [][]string  `json:"bids"`
	Asks         [][]string  `json:"asks"`
}

// tickerEvent represents the raw JSON structure from Binance individual symbol ticker stream.
type tickerEvent struct {
	EventType      string      `json:"e"`
	EventTime      json.Number `json:"E"`
	Symbol         string      `json:"s"`
	PriceChange    string      `json:"p"`
	PriceChangePct string      `json:"P"`
	WeightedAvg    string      `json:"w"`
	PrevClose      string      `json:"x"`
	Close          string      `json:"c"`
	CloseQty       string      `json:"Q"`
	Bid            string      `json:"b"`
	BidQty         string      `json:"B"`
	Ask            string      `json:"a"`
	AskQty         string      `json:"A"`
	Open           string      `json:"o"`
	High           string      `json:"h"`
	Low            string      `json:"l"`
	Volume         string      `json:"v"`
	QuoteVolume    string      `json:"q"`
	OpenTime       json.Number `json:"O"`
	CloseTime      json.Number `json:"C"`
	FirstID        json.Number `json:"F"`
	LastID         json.Number `json:"L"`
	Count          json.Number `json:"n"`
}

// NormalizeTrade converts a Binance trade event into the domain's Trade model.
func NormalizeTrade(raw []byte, eventID string) (market.Trade, error) {
	var ev tradeEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return market.Trade{}, err
	}
	if ev.EventType != "trade" {
		return market.Trade{}, fmt.Errorf("not a trade event: %s", ev.EventType)
	}

	side := market.SideBuy
	if ev.IsBuyerMK {
		side = market.SideSell
	}

	ts, _ := ev.EventTime.Int64()
	tid, _ := ev.TradeID.Int64()

	return market.Trade{
		EventID:         eventID,
		ExchangeTradeID: fmt.Sprintf("%d", tid),
		Exchange:        "binance",
		Pair:            ev.Symbol,
		Price:           ev.Price,
		Quantity:        ev.Quantity,
		Side:            side,
		TimestampUs:     ts * 1000,
	}, nil
}

// NormalizeOrderBook converts a Binance depth event into the domain's OrderBookUpdate model.
func NormalizeOrderBook(raw []byte, eventID string, stream string) (market.OrderBookUpdate, error) {
	var ev depthEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return market.OrderBookUpdate{}, err
	}
	seq, _ := ev.LastUpdateID.Int64()
	if seq == 0 {
		return market.OrderBookUpdate{}, fmt.Errorf("not a depth event")
	}

	// Extract symbol from stream name (e.g. btcusdt@depth20@100ms -> BTCUSDT)
	symbol := ""
	for i, r := range stream {
		if r == '@' {
			symbol = stream[:i]
			break
		}
	}
	// Convert to uppercase to match other events
	symbolUpper := ""
	for _, r := range symbol {
		if r >= 'a' && r <= 'z' {
			symbolUpper += string(r - ('a' - 'A'))
		} else {
			symbolUpper += string(r)
		}
	}

	bids := make([]market.PriceLevel, len(ev.Bids))
	for i, b := range ev.Bids {
		bids[i] = market.PriceLevel{Price: b[0], Quantity: b[1]}
	}
	asks := make([]market.PriceLevel, len(ev.Asks))
	for i, a := range ev.Asks {
		asks[i] = market.PriceLevel{Price: a[0], Quantity: a[1]}
	}

	return market.OrderBookUpdate{
		EventID:     eventID,
		Exchange:    "binance",
		Pair:        symbolUpper,
		Sequence:    seq,
		Bids:        bids,
		Asks:        asks,
		TimestampUs: time.Now().UnixNano() / 1000,
	}, nil
}

// NormalizeTicker converts a Binance ticker event into the domain's Ticker model.
func NormalizeTicker(raw []byte, eventID string) (market.Ticker, error) {
	var ev tickerEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return market.Ticker{}, err
	}
	if ev.EventType != "24hrTicker" {
		return market.Ticker{}, fmt.Errorf("not a ticker event: %s", ev.EventType)
	}
	ts, _ := ev.EventTime.Int64()

	return market.Ticker{
		EventID:     eventID,
		Exchange:    "binance",
		Pair:        ev.Symbol,
		Open:        ev.Open,
		High:        ev.High,
		Low:         ev.Low,
		Close:       ev.Close,
		Volume:      ev.Volume,
		QuoteVolume: ev.QuoteVolume,
		TimestampUs: ts * 1000,
	}, nil
}
