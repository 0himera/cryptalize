CREATE TABLE IF NOT EXISTS market.trades (
    event_id UUID,
    exchange LowCardinality(String),
    pair LowCardinality(String),
    price Decimal(38, 18),
    quantity Decimal(38, 18),
    side Enum8('BUY_SELL_UNSPECIFIED' = 0, 'BUY_SELL_BUY' = 1, 'BUY_SELL_SELL' = 2),
    timestamp_us DateTime64(6)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp_us)
ORDER BY (exchange, pair, timestamp_us);
