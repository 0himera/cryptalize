CREATE TABLE IF NOT EXISTS market.tickers (
    exchange LowCardinality(String),
    pair LowCardinality(String),
    open Decimal(38, 18),
    high Decimal(38, 18),
    low Decimal(38, 18),
    close Decimal(38, 18),
    volume Decimal(38, 18),
    quote_volume Decimal(38, 18),
    timestamp_us DateTime64(6),
    event_id UUID
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp_us)
ORDER BY (exchange, pair, timestamp_us);
