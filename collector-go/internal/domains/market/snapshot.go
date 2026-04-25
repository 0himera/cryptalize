package market

import (
	"sync"
)

// SnapshotStore keeps the latest market state in-memory for fast access.
type SnapshotStore struct {
	mu         sync.RWMutex
	tickers    map[string]*Ticker      // key: exchange:pair
	lastTrades map[string]*Trade       // key: exchange:pair
	connStatus map[string]bool         // key: exchange
}

func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{
		tickers:    make(map[string]*Ticker),
		lastTrades: make(map[string]*Trade),
		connStatus: make(map[string]bool),
	}
}

func (s *SnapshotStore) UpdateTicker(ticker *Ticker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ticker.Exchange + ":" + ticker.Pair
	s.tickers[key] = ticker
}

func (s *SnapshotStore) UpdateTrade(trade *Trade) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := trade.Exchange + ":" + trade.Pair
	s.lastTrades[key] = trade
}

func (s *SnapshotStore) SetConnStatus(exchange string, connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connStatus[exchange] = connected
}

func (s *SnapshotStore) GetTickers() map[string]*Ticker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to avoid race conditions during iteration
	copy := make(map[string]*Ticker)
	for k, v := range s.tickers {
		copy[k] = v
	}
	return copy
}

func (s *SnapshotStore) GetStatus() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[string]bool)
	for k, v := range s.connStatus {
		copy[k] = v
	}
	return copy
}
