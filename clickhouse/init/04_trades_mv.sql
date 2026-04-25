CREATE MATERIALIZED VIEW IF NOT EXISTS market.trades_mv TO market.trades AS
SELECT
    toUUIDOrDefault(`trade.event_id`) AS event_id,
    `trade.exchange` AS exchange,
    `trade.pair` AS pair,
    toDecimal128OrZero(`trade.price`, 18) AS price,
    toDecimal128OrZero(`trade.quantity`, 18) AS quantity,
    `trade.side` AS side,
    fromUnixTimestamp64Micro(`trade.timestamp_us`) AS timestamp_us
FROM market.events_queue
WHERE `trade.event_id` != '';
