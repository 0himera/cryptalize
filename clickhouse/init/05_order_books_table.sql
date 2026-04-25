CREATE TABLE IF NOT EXISTS market.order_books (
    exchange LowCardinality(String),
    pair LowCardinality(String),
    side Enum8('BID' = 1, 'ASK' = 2),
    price Decimal(38, 18),
    quantity Decimal(38, 18),
    sequence Int64,
    timestamp_us DateTime64(6),
    event_id UUID
) ENGINE = ReplacingMergeTree(sequence)
PARTITION BY toYYYYMMDD(timestamp_us)
ORDER BY (exchange, pair, side, price);
