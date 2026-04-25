-- We use ARRAY JOIN to flatten the nested bids/asks from the Protobuf message into individual rows.
CREATE MATERIALIZED VIEW IF NOT EXISTS market.order_books_mv TO market.order_books AS
SELECT
    exchange,
    pair,
    level.3 AS side,
    toDecimal128OrZero(level.1, 18) AS price,
    toDecimal128OrZero(level.2, 18) AS quantity,
    sequence,
    timestamp_us,
    event_id
FROM (
    SELECT
        `order_book_update.exchange` AS exchange,
        `order_book_update.pair` AS pair,
        `order_book_update.sequence` AS sequence,
        fromUnixTimestamp64Micro(`order_book_update.timestamp_us`) AS timestamp_us,
        toUUIDOrDefault(`order_book_update.event_id`) AS event_id,
        arrayMap(x -> (x.1, x.2, 1), `order_book_update.bids`) AS bids_raw,
        arrayMap(x -> (x.1, x.2, 2), `order_book_update.asks`) AS asks_raw,
        arrayConcat(bids_raw, asks_raw) AS all_levels
    FROM market.events_queue
    WHERE `order_book_update.event_id` != ''
)
ARRAY JOIN all_levels AS level
WHERE level.1 != ''; -- Skip empty price levels
