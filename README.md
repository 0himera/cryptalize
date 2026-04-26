# Architecture Overview

## Core Philosophy
Adherence to strict separation of Intent (domain logic) and Execution (infrastructure). Systems are assembled via explicit dependency injection. The architecture favors minimalistic, forward-leaning standards (Protobuf, ClickHouse over Kafka) to ensure predictability, universal traceability, and immutability of the core domain.

## Components

### 1. Collector (Data Ingestion Layer)
*   **Implementation:** Go Microservice.
*   **Role:** High-concurrency, low-latency market data ingestion.
*   **Mechanics:**
    *   Maintains asynchronous WebSocket connections to multiple cryptocurrency exchanges (Binance, Kraken).
    *   Normalizes disparate vendor-specific JSON payloads into a unified canonical schema (`market.v1` Protobuf).
    *   Publishes normalized events (`MarketEvent`) reliably to Kafka (Redpanda) via `franz-go`.
    *   Maintains an in-memory real-time state snapshot (latest prices, active connections) accessible via HTTP.
    *   Exposes a gRPC interface (`CollectorService`) for dynamic subscription management (e.g., add/remove trading pairs on the fly).
*   **Key Dependencies:** `franz-go` (Kafka), `nhooyr.io/websocket`, `google.golang.org/grpc`, `prometheus/client_golang`.

### 2. Event Streaming Layer
*   **Implementation:** Redpanda (Kafka API compatible).
*   **Role:** Durable, high-throughput message broker and buffer.
*   **Mechanics:**
    *   Decouples the fast ingestion layer from the storage/analytics layer.
    *   Guarantees zero data loss in the event of downstream sink unavailability.
    *   Topic: `market.events`. Message format: `ProtobufSingle`.

### 3. Analytical Storage (OLAP)
*   **Implementation:** ClickHouse.
*   **Role:** High-performance analytical querying over historical market data.
*   **Mechanics:**
    *   **Ingestion:** Native ClickHouse Kafka Engine (`market.events_queue`) consumes directly from Redpanda. Decodes Protobuf natively.
    *   **Transform:** Materialized Views (`trades_mv`, `order_books_mv`, `tickers_mv`) route and flatten data from the Kafka stream into structured MergeTree tables.
    *   **Engines utilized:**
        *   `MergeTree`: Append-only, time-partitioned historical logs (`trades`, `tickers`).
        *   `ReplacingMergeTree`: Idempotent state management for order books (uses sequence number to deduplicate and maintain current state).
        *   `AggregatingMergeTree`: Continuous background rollups for pre-calculated 1-minute OHLCV candles (`ohlcv_1m`), enabling sub-millisecond aggregate queries over massive datasets.

### 4. Brain (Analytics & Serving Layer)
*   **Implementation:** Python (FastAPI).
*   **Role:** Client-facing API and complex analytics engine.
*   **Mechanics:**
    *   Executes complex OLAP queries against ClickHouse via `clickhouse-connect`.
    *   Provides normalized REST API endpoints for downstream consumers (dashboards, algorithmic trading bots).
    *   Calculates real-time financial metrics (VWAP, Order Book Imbalance, Spreads, dynamic OHLCV aggregation).

### 5. Relational State (OLTP) / Cache
*   **PostgreSQL:** designated for slow, relational configuration data (user accounts, persistent configurations).
*   **Redis:** designated for 0-latency live state lookups (latest price caching).

## Data Flow
1. Vendor WS -> [Go Collector] -> Unify -> Serialize (Protobuf)
2. [Go Collector] -> Publish -> [Redpanda] (Topic: `market.events`)
3. [Redpanda] -> Consume -> [ClickHouse Kafka Engine] -> Parse Protobuf
4. [ClickHouse Kafka Engine] -> Materialized Views -> Flatten/Aggregate -> [MergeTree Tables]
5. Client HTTP -> [Python Brain] -> Query -> [ClickHouse] -> Respond JSON

## Serialization
*   **Protocol:** Protobuf v3.
*   **Precision:** High-precision financial metrics (Prices, Quantities) are transmitted as `string` types in Protobuf to bypass IEEE-754 floating-point inaccuracies. ClickHouse parses these strings into native `Decimal(38, 18)` types upon ingestion.