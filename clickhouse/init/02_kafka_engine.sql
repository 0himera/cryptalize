CREATE TABLE IF NOT EXISTS market.events_queue (
    `trade.event_id` String,
    `trade.exchange_trade_id` String,
    `trade.exchange` String,
    `trade.pair` String,
    `trade.price` String,
    `trade.quantity` String,
    `trade.side` Enum8('BUY_SELL_UNSPECIFIED' = 0, 'BUY_SELL_BUY' = 1, 'BUY_SELL_SELL' = 2),
    `trade.timestamp_us` Int64
) ENGINE = Kafka('redpanda:9092', 'market.events', 'clickhouse_consumer', 'ProtobufSingle')
SETTINGS 
    format_schema = 'market/v1/market.proto:MarketEvent',
    kafka_skip_broken_messages = 1;
