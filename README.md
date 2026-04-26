# Cryptalize

High-performance, multi-exchange market data pipeline and analytics API.

**Data flow**: Exchange WebSocket → Go Normalizer → Protobuf → Redpanda → ClickHouse MVs → FastAPI

---

## Architecture

```
┌──────────────┐    WS      ┌─────────────────┐  ProtobufSingle  ┌──────────────┐
│ Binance      ├───────────►│                 ├─────────────────►│              │
│ (BTC/USDT,   │            │  collector-go   │  topic:          │   Redpanda   │
│  ETH/USDT)   │            │  (Go 1.25)      │  market.events   │  (Kafka API) │
├──────────────┤            │                 │                  │              │
│ Kraken       ├───────────►│                 │                  └──────┬───────┘
│ (BTC/USD,    │            └─────────────────┘                         │
│  ETH/USD)    │                                               Kafka Engine
└──────────────┘                                                        │
                                                                        ▼
                                                          ┌─────────────────────────┐
                                                          │  ClickHouse             │
                                                          │  ┌───────────────────┐  │
                                                          │  │ market.events_queue│  │
                                                          │  │ (Kafka Engine)    │  │
                                                          │  └────────┬──────────┘  │
                                                          │           │ MVs          │
                                                          │  ┌────────┴──────────┐  │
                                                          │  │ market.trades      │  │
                                                          │  │ market.order_books │  │
                                                          │  │ market.ohlcv_1m    │  │
                                                          │  │ market.tickers     │  │
                                                          │  └───────────────────┘  │
                                                          └────────────┬────────────┘
                                                                       │ HTTP/8123
                                                                       ▼
                                                          ┌─────────────────────────┐
                                                          │  api-py (FastAPI)        │
                                                          │  :8001 → ORJSONResponse  │
                                                          └─────────────────────────┘
```

---

## Services

| Service             | Image / Build        | Ports                    | Role                                   |
|---------------------|----------------------|--------------------------|----------------------------------------|
| `redpanda`          | `redpandadata/redpanda:v23.2.1` | `19092` (external Kafka), `8082` (Pandaproxy) | Event bus |
| `clickhouse`        | `clickhouse/clickhouse-server:latest` | `8123` (HTTP), `9000` (native) | OLAP storage |
| `collector`         | `./collector-go` (Go 1.24+auto) | `9001` (gRPC), `9100` (Prometheus) | WebSocket ingestion |
| `brain`             | `./api-py` (Python 3.11) | `8001→8000`            | Analytics REST API |
| `prometheus`        | `prom/prometheus`    | `9090`                   | Metrics scraping |
| `postgres`          | `postgres:16-alpine` | `5432`                   | OLTP (future use) |
| `redis`             | `redis:7-alpine`     | `6379`                   | Cache / state (future use) |

---

## Collector — `collector-go`

**Language**: Go 1.25 (Docker uses `golang:1.24-alpine` + `GOTOOLCHAIN=auto`)  
**Module**: `github.com/0himera/cryptalize/collector-go`

### Process startup (port binding order)

1. Initialize Kafka publisher (`franz-go`, brokers from `KAFKA_BROKERS`)
2. Instantiate `SnapshotStore` (in-memory last-snapshot cache)
3. Build Binance + Kraken collectors and call `Subscribe(ctx, pair)` for each pair
4. Launch collectors in goroutines (`Run(ctx)`)
5. gRPC server on `:9001`
6. HTTP snapshot server on `:8000`
7. Prometheus metrics on `:9100`
8. Block on `SIGINT / SIGTERM` → `GracefulStop`

### Environment variables

| Variable        | Default              | Description                               |
|-----------------|----------------------|-------------------------------------------|
| `KAFKA_BROKERS` | `localhost:19092`    | Comma-separated broker list               |
| `KAFKA_TOPIC`   | `market.events`      | Target Kafka topic                        |
| `BINANCE_PAIRS` | `BTC/USDT,ETH/USDT`  | Comma-separated pairs (canonical `/` form)|
| `KRAKEN_PAIRS`  | `BTC/USD,ETH/USD`    | Comma-separated pairs                     |
| `GRPC_PORT`     | `9001`               | gRPC listen port                          |
| `METRICS_PORT`  | `9100`               | Prometheus scrape port                    |

### Domain models

#### `market.Trade`
```go
type Trade struct {
    EventID         string  // UUID v7, collector-assigned
    ExchangeTradeID string  // Exchange-native ID (may be empty)
    Exchange        string  // "binance" | "kraken"
    Pair            string  // Canonical: "BTC/USDT"
    Price           string  // Decimal string, preserves precision
    Quantity        string  // Decimal string
    Side            BuySell // SideUnspecified=0, SideBuy=1, SideSell=2
    TimestampUs     int64   // Unix microseconds (exchange-reported)
}
```

#### `market.OrderBookUpdate`
```go
type OrderBookUpdate struct {
    EventID     string       // UUID v7
    Exchange    string
    Pair        string
    Sequence    int64        // Exchange sequence; Kraken=UnixMicro(timestamp)
    Bids        []PriceLevel // Qty="0" means level deleted
    Asks        []PriceLevel
    TimestampUs int64
}

type PriceLevel struct {
    Price    string // Decimal string
    Quantity string // "0" = delete level
}
```

### Exchange normalizer behaviour

| Exchange | Stream            | Depth model       | Sequence source         | Notes |
|----------|-------------------|-------------------|-------------------------|-------|
| Binance  | `@depth20@100ms`  | Full snapshot (top-20) | Binance `lastUpdateId` | Stale prices not zeroed; query must filter `sequence = max(sequence)` |
| Kraken   | `book` (v2)       | Incremental delta | `UnixMicro(timestamp)`  | `qty=0` removes levels; query uses `FINAL` + 30 s temporal window |

---

## Kafka / Redpanda

**Topic**: `market.events`  
**Format**: `ProtobufSingle` (no framing bytes)  
**Schema**: `proto/market/v1/market.proto` → `MarketEvent`

### `MarketEvent` oneof

```proto
message MarketEvent {
  oneof payload {
    Trade            trade             = 1;
    OrderBookUpdate  order_book_update = 2;
    Ticker           ticker            = 3;
  }
}
```

Full field list:

| Message          | Field             | Type    | Notes |
|------------------|-------------------|---------|-------|
| `Trade`          | `event_id`        | string  | UUID v7 |
|                  | `exchange`        | string  | |
|                  | `pair`            | string  | |
|                  | `price`           | string  | Decimal |
|                  | `quantity`        | string  | Decimal |
|                  | `side`            | BuySell | enum |
|                  | `timestamp_us`    | int64   | Unix µs |
| `OrderBookUpdate`| `event_id`        | string  | |
|                  | `sequence`        | int64   | |
|                  | `bids` / `asks`   | repeated PriceLevel | |
|                  | `timestamp_us`    | int64   | |
| `Ticker`         | `open/high/low/close/volume/quote_volume` | string | Decimal |
|                  | `timestamp_us`    | int64   | |

---

## ClickHouse Schema

### `market.events_queue` (Kafka Engine)

Flat projection of `MarketEvent`. Protobuf nested repeated messages map to **parallel arrays**:

```sql
`order_book_update.bids.price`    Array(String)
`order_book_update.bids.quantity` Array(String)
`order_book_update.asks.price`    Array(String)
`order_book_update.asks.quantity` Array(String)
```

> Using `Array(Tuple(...))` fails silently with ClickHouse ProtobufSingle deserialization. Parallel arrays are the correct mapping.

### `market.trades`

| Column        | Type              | Notes |
|---------------|-------------------|-------|
| `event_id`    | UUID              | |
| `exchange`    | LowCardinality(String) | |
| `pair`        | LowCardinality(String) | |
| `price`       | Decimal(38, 18)   | |
| `quantity`    | Decimal(38, 18)   | |
| `side`        | Enum8             | |
| `timestamp_us`| DateTime64(6)     | |

Engine: `MergeTree()`, `ORDER BY (exchange, pair, timestamp_us)`

### `market.order_books`

Engine: `ReplacingMergeTree(sequence)`  
`ORDER BY (exchange, pair, side, price)`

| Column        | Type              | Notes |
|---------------|-------------------|-------|
| `side`        | Enum8('BID'=1,'ASK'=2) | |
| `price`       | Decimal(38, 18)   | Sort key component |
| `quantity`    | Decimal(38, 18)   | `0` = deleted level |
| `sequence`    | Int64             | Version column for ReplacingMT |
| `timestamp_us`| DateTime64(6)     | Used for 30 s staleness filter |

**MV logic**: `order_books_mv` uses `arrayMap((p,q)->(p,q,side), bids.price, bids.quantity)` + `arrayConcat` + `ARRAY JOIN` to flatten parallel arrays into per-level rows.

### `market.ohlcv_1m`

Engine: `AggregatingMergeTree()`, `ORDER BY (exchange, pair, bucket)`

| Column                   | Type                                        | Notes |
|--------------------------|---------------------------------------------|-------|
| `bucket`                 | DateTime (1 min)                            | `toStartOfMinute(timestamp_us)` |
| `open_state`             | AggregateFunction(argMin, Decimal(38,18), DateTime64(6)) | |
| `close_state`            | AggregateFunction(argMax, ...)              | |
| `cumulative_pv_state`    | AggregateFunction(sum, Decimal(76,36))      | VWAP numerator |
| `cumulative_volume_state`| AggregateFunction(sum, Decimal(38,18))      | VWAP denominator |
| `quote_volume`           | SimpleAggregateFunction(sum, Decimal(76,36))| High-precision to avoid overflow at BTC-scale prices |

**VWAP precision note**: `price * quantity` where both are `Decimal(38,18)` produces `Decimal(76,36)`. The result must be explicitly cast before `sumState` to avoid ClickHouse type-mismatch errors.

---

## gRPC API — `CollectorService`

**Proto package**: `collector.v1`  
**Port**: `:9001`

```proto
service CollectorService {
  rpc Subscribe   (SubscribeRequest)   returns (SubscribeResponse);
  rpc Unsubscribe (UnsubscribeRequest) returns (UnsubscribeResponse);
}

message SubscribeRequest   { string exchange = 1; string pair = 2; }
message SubscribeResponse  { bool success = 1; string message = 2; }
message UnsubscribeRequest { string exchange = 1; string pair = 2; }
message UnsubscribeResponse{ bool success = 1; string message = 2; }
```

---

## Running Locally

### Prerequisites

- Docker + Docker Compose
- Go ≥ 1.24 (auto-toolchain will fetch 1.25 if needed)

### Start the stack

```bash
docker compose up -d
```

This starts Redpanda, ClickHouse (with DDL init), Postgres, Redis, Prometheus, the collector, and the brain in dependency order.

### Run collector on host (dev)

```bash
cd collector-go
KAFKA_BROKERS=localhost:19092 go run ./shell/server/
```

### Query the API

```bash
# Full summary (spread, VWAP, imbalance, 24h stats)
curl localhost:8001/analytics/summary/binance/BTCUSDT
curl localhost:8001/analytics/summary/kraken/BTC/USD   # slash in pair handled via :path

# Raw order book
curl localhost:8001/orderbook/binance/BTCUSDT

# OHLCV candles
curl "localhost:8001/analytics/ohlcv/binance/BTCUSDT?interval=5m&limit=50"
```

### ClickHouse direct

```bash
docker exec -it cryptalize-clickhouse \
  clickhouse-client --user admin --password admin
```

---

## Prometheus Metrics

Collector exposes Go runtime + custom metrics at `http://collector:9100/metrics`.

Prometheus config: `./prometheus/prometheus.yml`

---

## Repository Layout

```
cryptalize/
├── collector-go/           # Go ingestion service
│   ├── internal/
│   │   ├── app/            # Config (env vars)
│   │   ├── domains/market/ # Domain types (Trade, OrderBookUpdate, Ticker)
│   │   ├── infra/
│   │   │   ├── exchange/   # binance/, kraken/ — WS clients + normalizers
│   │   │   ├── kafka/      # franz-go publisher
│   │   │   └── metrics/    # Prometheus counters
│   │   ├── protocol/
│   │   │   ├── grpcapi/    # CollectorService implementation
│   │   │   └── httpapi/    # Snapshot HTTP handler
│   │   └── proto/          # Generated Go protobuf bindings
│   └── shell/server/       # main.go — wiring + startup
├── api-py/                 # FastAPI analytics brain
│   ├── main.py             # All routes
│   └── requirements.txt
├── clickhouse/init/        # DDL executed at container start
│   ├── 01_database.sql
│   ├── 02_kafka_engine.sql # events_queue (Kafka Engine)
│   ├── 03_trades_table.sql
│   ├── 04_trades_mv.sql
│   ├── 05_order_books_table.sql
│   ├── 06_order_books_mv.sql
│   ├── 07_tickers_table.sql
│   ├── 08_tickers_mv.sql
│   ├── 09_ohlcv_1m_table.sql
│   └── 10_ohlcv_1m_mv.sql
├── proto/                  # .proto source files (also mounted into ClickHouse)
│   ├── market/v1/market.proto
│   └── collector/v1/collector.proto
├── prometheus/             # prometheus.yml
└── docker-compose.yml
```
