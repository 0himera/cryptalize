# Cryptalize API Reference

Base URL: `http://localhost:8001`  
Serializer: `orjson` (all responses `Content-Type: application/json`)  
Pair with `/` in name: use URL path as-is (`:path` routing, e.g. `/analytics/summary/kraken/BTC/USD`)

---

## `GET /`

Health probe.

**Response**
```json
{ "message": "Cryptalize Brain is active" }
```

---

## Market Data

### `GET /trades/{exchange}/{pair}`

Last N raw trades from `market.trades`, ordered newest-first.

**Path params**

| Param      | Type   | Example    |
|------------|--------|------------|
| `exchange` | string | `binance`  |
| `pair`     | string | `BTCUSDT`  |

**Query params**

| Param   | Default | Description         |
|---------|---------|---------------------|
| `limit` | `100`   | Max rows to return  |

**Response** `200 Trade[]`
```json
[
  {
    "event_id":     "019dc6cf-7682-7434-9c7d-66721a07c3fe",
    "exchange":     "binance",
    "pair":         "BTCUSDT",
    "price":        77610.9,
    "quantity":     0.02382462,
    "side":         "BUY_SELL_BUY",
    "timestamp_us": "2026-04-26T11:42:58.259630"
  }
]
```

| Field          | Type     | Notes                              |
|----------------|----------|------------------------------------|
| `event_id`     | string   | UUID v7, collector-generated       |
| `exchange`     | string   |                                    |
| `pair`         | string   |                                    |
| `price`        | float64  | Cast from Decimal128               |
| `quantity`     | float64  |                                    |
| `side`         | string   | `BUY_SELL_BUY` \| `BUY_SELL_SELL` \| `BUY_SELL_UNSPECIFIED` |
| `timestamp_us` | datetime | ISO 8601, from ClickHouse DateTime64(6) |

---

### `GET /orderbook/{exchange}/{pair}`

Current reconstructed order book from `market.order_books FINAL`, all active levels.

> **Note**: Does not apply the 30 s staleness filter. Use `/analytics/spread` for fresh-data-only best bid/ask.

**Response** `200`
```json
{
  "exchange": "kraken",
  "pair":     "BTC/USD",
  "bids": [
    { "price": 78086.9, "quantity": 0.41 }
  ],
  "asks": [
    { "price": 78087.0, "quantity": 0.12 }
  ]
}
```

Levels sorted by price descending (bids highest first, asks also in descending order).

---

## Analytics

### `GET /analytics/ohlcv/{exchange}/{pair}`

OHLCV candles aggregated from `market.ohlcv_1m`.

**Query params**

| Param      | Default | Allowed                            |
|------------|---------|------------------------------------|
| `interval` | `1m`    | `1m`, `5m`, `15m`, `1h`, `4h`, `1d` |
| `limit`    | `100`   | Max candles                        |

**Response** `200 Candle[]`
```json
[
  {
    "bucket":       "2026-04-26T11:42:00",
    "open":         78060.0,
    "high":         78110.5,
    "low":          78050.1,
    "close":        78087.0,
    "volume":       1.4352,
    "quote_volume": 112031.44,
    "trade_count":  87
  }
]
```

Ordered newest-first.

| Field         | Type     | Notes                                                    |
|---------------|----------|----------------------------------------------------------|
| `bucket`      | datetime | Floor of the interval                                    |
| `open`        | float64  | `argMin(price, timestamp_us)` merged across 1 m buckets  |
| `close`       | float64  | `argMax(price, timestamp_us)` merged                     |
| `quote_volume`| float64  | Sum of `price × quantity` per trade (Decimal76 precision) |
| `trade_count` | int      |                                                          |

---

### `GET /analytics/vwap/{exchange}/{pair}`

Cumulative daily VWAP (resets at `toStartOfDay(now())`, UTC).

**Calculation**
```sql
sumMerge(cumulative_pv_state) / nullIf(sumMerge(cumulative_volume_state), 0)
WHERE bucket >= toStartOfDay(now())
```

**Response** `200`
```json
{
  "exchange":  "binance",
  "pair":      "BTCUSDT",
  "vwap":      78077.413,
  "timestamp": "2026-04-26T11:39:58.687938"
}
```

Returns `vwap: 0.0` when no trade data exists for today.

---

### `GET /analytics/spread/{exchange}/{pair}`

Best bid / ask and derived spread metrics.

**Exchange-specific query logic**

| Exchange    | Strategy                                                          |
|-------------|-------------------------------------------------------------------|
| `binance`   | Filter `sequence = max(sequence)` — isolates the latest snapshot (avoids stale levels from prior partial snapshots) |
| others      | `FINAL` + `timestamp_us > now() - INTERVAL 30 SECOND` — reconstructs current incremental book |

**Response** `200`
```json
{
  "exchange":   "kraken",
  "pair":       "BTC/USD",
  "bid":        78086.9,
  "ask":        78087.0,
  "spread":     0.1,
  "spread_bps": 0.01280
}
```

| Field        | Type    | Formula                              |
|--------------|---------|--------------------------------------|
| `spread`     | float64 | `ask - bid`                          |
| `spread_bps` | float64 | `(spread / mid) × 10000`             |

**Error** `404` when order book has no data within the lookback window.

---

### `GET /analytics/imbalance/{exchange}/{pair}`

Volume-weighted order book pressure ratio.

**Calculation**
```
imbalance = sum(bid_qty) / (sum(bid_qty) + sum(ask_qty))
```

- `1.0` → all volume on bid side (maximum buy pressure)
- `0.5` → neutral
- `0.0` → all volume on ask side (maximum sell pressure)

Same exchange branching as `/analytics/spread` (snapshot vs. incremental + 30 s window).

**Query params**

| Param   | Default | Notes                          |
|---------|---------|--------------------------------|
| `depth` | `10`    | Reserved; currently uses all active levels |

**Response** `200`
```json
{
  "exchange":        "binance",
  "pair":            "BTCUSDT",
  "imbalance_ratio": 0.187
}
```

Returns `imbalance_ratio: 0.5` when no data available.

---

### `GET /analytics/summary/{exchange}/{pair}`

Composite endpoint. Single call returns spread + VWAP + imbalance + 24 h stats. Intended for dashboard polling.

**Internally calls** (sequentially): `get_spread`, `get_vwap`, `get_imbalance`, plus one additional ClickHouse query for 24 h open/close/volume.

**Response** `200`
```json
{
  "exchange":       "kraken",
  "pair":           "BTC/USD",
  "spread": {
    "exchange":   "kraken",
    "pair":       "BTC/USD",
    "bid":        78086.9,
    "ask":        78087.0,
    "spread":     0.1,
    "spread_bps": 0.01280
  },
  "vwap":           78087.0,
  "imbalance":      0.048,
  "last_price":     78087.0,
  "change_24h_pct": 0.627,
  "volume_24h":     2.187,
  "timestamp":      "2026-04-26T11:39:45.668036"
}
```

| Field            | Type    | Notes                                           |
|------------------|---------|-------------------------------------------------|
| `spread`         | object  | Full `SpreadResponse` (see above)               |
| `vwap`           | float64 | Daily cumulative                                |
| `imbalance`      | float64 | 0→1 ratio                                       |
| `last_price`     | float64 | `argMax(price, timestamp_us)` over last 24 h    |
| `change_24h_pct` | float64 | `(last - open) / open × 100`                    |
| `volume_24h`     | float64 | Base asset volume over last 24 h                |

Fields `last_price`, `change_24h_pct`, `volume_24h` absent when no trade data exists.

**Error** `500` propagates inner exception detail as string.

---

## Error Codes

| Code | Condition                                              |
|------|--------------------------------------------------------|
| 404  | Order book data not found for exchange/pair            |
| 500  | Unhandled internal error (detail string in body)       |

---

## Environment — `api-py`

| Variable              | Default       |
|-----------------------|---------------|
| `CLICKHOUSE_HOST`     | `localhost`   |
| `CLICKHOUSE_PORT`     | `8123`        |
| `CLICKHOUSE_USER`     | `admin`       |
| `CLICKHOUSE_PASSWORD` | `admin`       |
