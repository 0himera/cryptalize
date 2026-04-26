# Cryptalize API Documentation

This document outlines the exposed endpoints for the Cryptalize system, divided into the **Brain (Python/FastAPI) HTTP API**, the **Collector (Go) HTTP API**, and the **Collector (Go) gRPC Service**.

---

## 1. Brain API (Python FastAPI)

**Base URL:** `http://localhost:8001` (default docker mapping)
**Role:** Read-heavy analytics, serving historical data and complex aggregations from ClickHouse.

### 1.1 Market Data Endpoints

#### `GET /trades/{exchange}/{pair}`
Retrieve the most recent historical trades for a given exchange and pair.
*   **Parameters:**
    *   `exchange` (path): String (e.g., `binance`, `kraken`).
    *   `pair` (path): String (e.g., `BTC/USDT`).
    *   `limit` (query): Integer. Default `100`. Number of trades to return.
*   **Response:** `200 OK`
    *   List of `Trade` objects.
    *   ```json
        [
          {
            "event_id": "uuid",
            "exchange": "binance",
            "pair": "BTCUSDT",
            "price": 60000.0,
            "quantity": 0.1,
            "side": "BUY_SELL_SELL",
            "timestamp_us": "2023-01-01T00:00:00Z"
          }
        ]
        ```

#### `GET /orderbook/{exchange}/{pair}`
Retrieve the reconstructed order book state.
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
*   **Response:** `200 OK`
    *   ```json
        {
          "exchange": "binance",
          "pair": "BTCUSDT",
          "bids": [{"price": 59999.0, "quantity": 1.5}, ...],
          "asks": [{"price": 60001.0, "quantity": 0.5}, ...]
        }
        ```

### 1.2 Analytics Endpoints

#### `GET /analytics/ohlcv/{exchange}/{pair}`
Retrieve historical OHLCV (Open, High, Low, Close, Volume) candles aggregated dynamically.
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
    *   `interval` (query): String matching regex `^(1m|5m|15m|1h|4h|1d)$`. Default `1m`.
    *   `limit` (query): Integer. Default `100`.
*   **Response:** `200 OK`
    *   List of `Candle` objects.

#### `GET /analytics/vwap/{exchange}/{pair}`
Calculate the cumulative Daily Volume-Weighted Average Price (VWAP).
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
*   **Response:** `200 OK`
    *   `{"exchange": "binance", "pair": "BTCUSDT", "vwap": 60100.5, "timestamp": "..."}`

#### `GET /analytics/spread/{exchange}/{pair}`
Calculate the current bid-ask spread and spread in Basis Points (BPS).
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
*   **Response:** `200 OK`
    *   `{"exchange": "binance", "pair": "...", "bid": 60000, "ask": 60001, "spread": 1.0, "spread_bps": 0.16}`

#### `GET /analytics/imbalance/{exchange}/{pair}`
Calculate the order book imbalance ratio ($BidVol / (BidVol + AskVol)$).
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
    *   `depth` (query): Integer. Default `10`.
*   **Response:** `200 OK`
    *   `{"exchange": "binance", "pair": "...", "imbalance_ratio": 0.55}`

#### `GET /analytics/summary/{exchange}/{pair}`
Aggregate snapshot combining spread, VWAP, imbalance, and 24h trailing metrics.
*   **Parameters:**
    *   `exchange` (path): String.
    *   `pair` (path): String.
*   **Response:** `200 OK`
    *   ```json
        {
          "exchange": "...",
          "pair": "...",
          "spread": {...},
          "vwap": 60100.5,
          "imbalance": 0.55,
          "change_24h_pct": 2.5,
          "volume_24h": 1500.0,
          "last_price": 60500.0,
          "timestamp": "..."
        }
        ```

---

## 2. Collector API (Go HTTP)

**Base URL:** `http://localhost:8000` (internal network routing)
**Role:** In-memory state inspection, health monitoring, and liveness probes.

### 2.1 State Endpoints

#### `GET /snapshot/tickers`
Retrieves the latest observed in-memory ticker objects across all active streams.
*   **Response:** `200 OK` returning JSON map of active tickers.

#### `GET /snapshot/orderbooks`
Retrieves the latest observed partial depth updates in memory.
*   **Response:** `200 OK` returning JSON map of active orderbook updates.

#### `GET /snapshot/status`
Retrieves active WebSocket connection statuses.
*   **Response:** `200 OK` returning JSON map of exchange connection booleans (e.g. `{"binance": true, "kraken": true}`).

### 2.2 System Endpoints

#### `GET /`
Retrieves current server unix seconds (Datetime Service).

#### `GET /healthz`
Kubernetes/Docker liveness probe endpoint.
*   **Response:** `200 OK` (Empty body)

#### `GET /metrics` (Port `9100`)
Prometheus metrics scraping endpoint.

---

## 3. Collector API (Go gRPC)

**Host:** `localhost:9001` (internal network routing)
**Role:** Dynamic control plane for the data ingestion layer.
**Service Definition:** `collector.v1.CollectorService`

### `rpc Subscribe(SubscribeRequest) returns (SubscribeResponse);`
Dynamically command the Collector to open a WebSocket channel for a new pair.
*   **Request:** `{"exchange": "binance", "pair": "ETH/USDT"}`
*   **Response:** `{"success": true, "message": "Subscribed to ETH/USDT on binance"}`

### `rpc Unsubscribe(UnsubscribeRequest) returns (UnsubscribeResponse);`
Dynamically command the Collector to drop a WebSocket channel.
*   **Request:** `{"exchange": "binance", "pair": "ETH/USDT"}`
*   **Response:** `{"success": true, "message": "Unsubscribed from ETH/USDT on binance"}`
