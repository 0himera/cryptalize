CREATE TABLE IF NOT EXISTS market.ohlcv_1m (
    exchange LowCardinality(String),
    pair LowCardinality(String),
    bucket DateTime,
    open_state AggregateFunction(argMin, Decimal(38, 18), DateTime64(6)),
    close_state AggregateFunction(argMax, Decimal(38, 18), DateTime64(6)),
    high SimpleAggregateFunction(max, Decimal(38, 18)),
    low SimpleAggregateFunction(min, Decimal(38, 18)),
    volume SimpleAggregateFunction(sum, Decimal(38, 18)),
    quote_volume SimpleAggregateFunction(sum, Decimal(76, 36)),
    cumulative_pv_state AggregateFunction(sum, Decimal(76, 36)),
    cumulative_volume_state AggregateFunction(sum, Decimal(38, 18)),
    trade_count SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMMDD(bucket)
ORDER BY (exchange, pair, bucket);
