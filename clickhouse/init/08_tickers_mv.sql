CREATE MATERIALIZED VIEW IF NOT EXISTS market.tickers_mv TO market.tickers AS
SELECT
    `ticker.exchange` AS exchange,
    `ticker.pair` AS pair,
    toDecimal128OrZero(`ticker.open`, 18) AS open,
    toDecimal128OrZero(`ticker.high`, 18) AS high,
    toDecimal128OrZero(`ticker.low`, 18) AS low,
    toDecimal128OrZero(`ticker.close`, 18) AS close,
    toDecimal128OrZero(`ticker.volume`, 18) AS volume,
    toDecimal128OrZero(`ticker.quote_volume`, 18) AS quote_volume,
    fromUnixTimestamp64Micro(`ticker.timestamp_us`) AS timestamp_us,
    toUUIDOrDefault(`ticker.event_id`) AS event_id
FROM market.events_queue
WHERE `ticker.event_id` != '';
