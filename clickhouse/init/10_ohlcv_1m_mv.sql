CREATE MATERIALIZED VIEW IF NOT EXISTS market.ohlcv_1m_mv TO market.ohlcv_1m AS
SELECT
    exchange,
    pair,
    toStartOfMinute(timestamp_us) AS bucket,
    argMinState(price, timestamp_us) AS open_state,
    argMaxState(price, timestamp_us) AS close_state,
    max(price) AS high,
    min(price) AS low,
    sum(quantity) AS volume,
    sum(price * quantity) AS quote_volume,
    sumState(CAST(price * quantity, 'Decimal(76, 36)')) AS cumulative_pv_state,
    sumState(quantity) AS cumulative_volume_state,
    count() AS trade_count
FROM market.trades
GROUP BY exchange, pair, bucket;
