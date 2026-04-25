CREATE TABLE IF NOT EXISTS market.events_queue (
    `trade.event_id` String,
    `trade.exchange_trade_id` String,
    `trade.exchange` String,
    `trade.pair` String,
    `trade.price` String,
    `trade.quantity` String,
    `trade.side` Enum8('BUY_SELL_UNSPECIFIED' = 0, 'BUY_SELL_BUY' = 1, 'BUY_SELL_SELL' = 2),
    `trade.timestamp_us` Int64,
    
    `order_book_update.event_id` String,
    `order_book_update.exchange` String,
    `order_book_update.pair` String,
    `order_book_update.sequence` Int64,
    `order_book_update.timestamp_us` Int64,
    `order_book_update.bids` Array(Tuple(price String, quantity String)),
    `order_book_update.asks` Array(Tuple(price String, quantity String)),
    
    `ticker.event_id` String,
    `ticker.exchange` String,
    `ticker.pair` String,
    `ticker.open` String,
    `ticker.high` String,
    `ticker.low` String,
    `ticker.close` String,
    `ticker.volume` String,
    `ticker.quote_volume` String,
    `ticker.timestamp_us` Int64
) ENGINE = Kafka('redpanda:9092', 'market.events', 'clickhouse_consumer', 'ProtobufSingle')
SETTINGS 
    format_schema = 'market/v1/market.proto:MarketEvent',
    kafka_skip_broken_messages = 1;
