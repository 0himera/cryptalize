from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import ORJSONResponse
from pydantic import BaseModel
from typing import List, Optional, Dict
import clickhouse_connect
from datetime import datetime, timedelta
import os

app = FastAPI(
    title="Cryptalize Analytics API",
    default_response_class=ORJSONResponse,
    description="Streaming Analytics & Observability API"
)

# ClickHouse Client
CH_HOST = os.getenv("CLICKHOUSE_HOST", "localhost")
CH_PORT = int(os.getenv("CLICKHOUSE_PORT", 8123))
CH_USER = os.getenv("CLICKHOUSE_USER", "admin")
CH_PASS = os.getenv("CLICKHOUSE_PASSWORD", "admin")

client = clickhouse_connect.get_client(
    host=CH_HOST, 
    port=CH_PORT, 
    username=CH_USER, 
    password=CH_PASS
)

# --- Models ---

class Trade(BaseModel):
    event_id: str
    exchange: str
    pair: str
    price: float
    quantity: float
    side: str
    timestamp_us: datetime

class Candle(BaseModel):
    bucket: datetime
    open: float
    high: float
    low: float
    close: float
    volume: float
    quote_volume: float
    trade_count: int

class VWAPResponse(BaseModel):
    exchange: str
    pair: str
    vwap: float
    timestamp: datetime

class SpreadResponse(BaseModel):
    exchange: str
    pair: str
    bid: float
    ask: float
    spread: float
    spread_bps: float

class ImbalanceResponse(BaseModel):
    exchange: str
    pair: str
    imbalance_ratio: float  # 0 to 1

# --- Routes ---

@app.get("/")
async def root():
    return {"message": "Cryptalize Brain is active"}

# 1. Market Data Endpoints

@app.get("/trades/{exchange}/{pair}", response_model=List[Trade])
async def get_trades(exchange: str, pair: str, limit: int = 100):
    query = """
        SELECT event_id, exchange, pair, 
               toFloat64(price) as price, 
               toFloat64(quantity) as quantity, 
               side, timestamp_us
        FROM market.trades
        WHERE exchange = %s AND pair = %s
        ORDER BY timestamp_us DESC
        LIMIT %s
    """
    result = client.query(query, [exchange, pair, limit])
    
    trades = []
    for row in result.result_rows:
        trades.append(Trade(
            event_id=str(row[0]),
            exchange=row[1],
            pair=row[2],
            price=row[3],
            quantity=row[4],
            side=row[5],
            timestamp_us=row[6]
        ))
    return trades

@app.get("/orderbook/{exchange}/{pair}")
async def get_orderbook(exchange: str, pair: str):
    query = """
        SELECT side, toFloat64(price), toFloat64(quantity)
        FROM market.order_books
        FINAL
        WHERE exchange = %s AND pair = %s AND quantity > 0
        ORDER BY price DESC
    """
    result = client.query(query, [exchange, pair])
    
    bids = []
    asks = []
    for row in result.result_rows:
        level = {"price": row[1], "quantity": row[2]}
        if row[0] == 'BID':
            bids.append(level)
        else:
            asks.append(level)
            
    return {
        "exchange": exchange,
        "pair": pair,
        "bids": bids,
        "asks": asks
    }

# 2. Analytics Endpoints

@app.get("/analytics/ohlcv/{exchange}/{pair}", response_model=List[Candle])
async def get_ohlcv(
    exchange: str, 
    pair: str, 
    interval: str = Query("1m", regex="^(1m|5m|15m|1h|4h|1d)$"),
    limit: int = 100
):
    # Map interval to ClickHouse toStartOfInterval
    # Since we have a 1m table, we can aggregate from it
    interval_map = {
        "1m": "1 minute",
        "5m": "5 minute",
        "15m": "15 minute",
        "1h": "1 hour",
        "4h": "4 hour",
        "1d": "1 day"
    }
    ch_interval = interval_map[interval]

    query = f"""
        SELECT 
            toStartOfInterval(bucket, INTERVAL {ch_interval}) as time,
            argMinMerge(open_state) as open,
            max(high) as high,
            min(low) as low,
            argMaxMerge(close_state) as close,
            sum(volume) as volume,
            sum(quote_volume) as quote_volume,
            sum(trade_count) as trade_count
        FROM market.ohlcv_1m
        WHERE exchange = %s AND pair = %s
        GROUP BY time
        ORDER BY time DESC
        LIMIT %s
    """
    result = client.query(query, [exchange, pair, limit])
    
    candles = []
    for row in result.result_rows:
        candles.append(Candle(
            bucket=row[0],
            open=float(row[1]),
            high=float(row[2]),
            low=float(row[3]),
            close=float(row[4]),
            volume=float(row[5]),
            quote_volume=float(row[6]),
            trade_count=int(row[7])
        ))
    return candles

@app.get("/analytics/vwap/{exchange}/{pair:path}", response_model=VWAPResponse)
async def get_vwap(exchange: str, pair: str):
    # Daily Cumulative VWAP
    query = """
        SELECT 
            sumMerge(cumulative_pv_state) / nullIf(sumMerge(cumulative_volume_state), 0) as vwap
        FROM market.ohlcv_1m
        WHERE exchange = %s AND pair = %s AND bucket >= toStartOfDay(now())
    """
    result = client.query(query, [exchange, pair])
    if not result.result_rows or result.result_rows[0][0] is None:
        return VWAPResponse(exchange=exchange, pair=pair, vwap=0.0, timestamp=datetime.now())
    
    return VWAPResponse(
        exchange=exchange,
        pair=pair,
        vwap=float(result.result_rows[0][0]),
        timestamp=datetime.now()
    )

@app.get("/analytics/spread/{exchange}/{pair:path}", response_model=SpreadResponse)
async def get_spread(exchange: str, pair: str):
    if exchange == "binance":
        # Binance partial depth (@depth20) sends full snapshots of the top 20 levels.
        # We must only look at the latest sequence to avoid stale levels.
        query = """
            WITH (SELECT max(sequence) FROM market.order_books WHERE exchange = %s AND pair = %s) as last_seq
            SELECT 
                maxIf(toFloat64(price), side = 'BID') as best_bid,
                minIf(toFloat64(price), side = 'ASK') as best_ask
            FROM market.order_books
            FINAL
            WHERE exchange = %s AND pair = %s AND sequence = last_seq AND quantity > 0
        """
        result = client.query(query, [exchange, pair, exchange, pair])
    else:
        # Kraken and others send incremental deltas (with qty=0 for deletions).
        # We must look across all sequences and rely on ReplacingMergeTree (FINAL) to give us the current state.
        query = """
            SELECT 
                maxIf(toFloat64(price), side = 'BID') as best_bid,
                minIf(toFloat64(price), side = 'ASK') as best_ask
            FROM market.order_books
            FINAL
            WHERE exchange = %s AND pair = %s AND quantity > 0
              AND timestamp_us > now() - INTERVAL 30 SECOND
        """
        result = client.query(query, [exchange, pair])
    if not result.result_rows or result.result_rows[0][0] is None or result.result_rows[0][1] is None:
        raise HTTPException(status_code=404, detail="Order book data not found")
    
    bid = result.result_rows[0][0]
    ask = result.result_rows[0][1]
    spread = ask - bid
    mid = (ask + bid) / 2
    bps = (spread / mid) * 10000 if mid > 0 else 0
    
    return SpreadResponse(
        exchange=exchange,
        pair=pair,
        bid=bid,
        ask=ask,
        spread=spread,
        spread_bps=bps
    )

@app.get("/analytics/imbalance/{exchange}/{pair:path}", response_model=ImbalanceResponse)
async def get_imbalance(exchange: str, pair: str, depth: int = 10):
    # Imbalance = Sum(BidQty) / (Sum(BidQty) + Sum(AskQty))
    # 0.5 is neutral. > 0.5 is bullish (more buy pressure). < 0.5 is bearish.
    query = """
        SELECT 
            side, sum(toFloat64(quantity)) as total_qty
        FROM (
            SELECT side, price, quantity
            FROM market.order_books
            FINAL
            WHERE exchange = %s AND pair = %s AND quantity > 0
            ORDER BY price DESC
            LIMIT %s  -- This is tricky because we need top N per side. 
                      -- For simplicity we'll just take everything in the partial snapshot
        )
        GROUP BY side
    """
    if exchange == "binance":
        # Ratio for latest available snapshot
        query = """
            WITH (SELECT max(sequence) FROM market.order_books WHERE exchange = %s AND pair = %s) as last_seq
            SELECT 
                sumIf(toFloat64(quantity), side = 'BID') as bid_vol,
                sumIf(toFloat64(quantity), side = 'ASK') as ask_vol
            FROM market.order_books
            FINAL
            WHERE exchange = %s AND pair = %s AND sequence = last_seq AND quantity > 0
        """
        result = client.query(query, [exchange, pair, exchange, pair])
    else:
        # Ratio across all active levels
        query = """
            SELECT 
                sumIf(toFloat64(quantity), side = 'BID') as bid_vol,
                sumIf(toFloat64(quantity), side = 'ASK') as ask_vol
            FROM market.order_books
            FINAL
            WHERE exchange = %s AND pair = %s AND quantity > 0
              AND timestamp_us > now() - INTERVAL 30 SECOND
        """
        result = client.query(query, [exchange, pair])
    if not result.result_rows or result.result_rows[0][0] is None:
        return ImbalanceResponse(exchange=exchange, pair=pair, imbalance_ratio=0.5)
    
    bid_vol = result.result_rows[0][0]
    ask_vol = result.result_rows[0][1]
    total = bid_vol + ask_vol
    ratio = bid_vol / total if total > 0 else 0.5
    
    return ImbalanceResponse(
        exchange=exchange,
        pair=pair,
        imbalance_ratio=ratio
    )

@app.get("/analytics/summary/{exchange}/{pair:path}")
async def get_summary(exchange: str, pair: str):
    # This combines multiple metrics into one call for the UI
    try:
        spread = await get_spread(exchange, pair)
        vwap = await get_vwap(exchange, pair)
        imbalance = await get_imbalance(exchange, pair)
        
        # Get last 24h change and volume
        query = """
            SELECT 
                argMinMerge(open_state) as open_24h,
                argMaxMerge(close_state) as last_price,
                sum(volume) as vol_24h
            FROM market.ohlcv_1m
            WHERE exchange = %s AND pair = %s AND bucket >= now() - INTERVAL 24 hour
        """
        res = client.query(query, [exchange, pair])
        
        summary = {
            "exchange": exchange,
            "pair": pair,
            "spread": spread,
            "vwap": vwap.vwap,
            "imbalance": imbalance.imbalance_ratio,
            "timestamp": datetime.now()
        }
        
        if res.result_rows and res.result_rows[0][0] is not None:
            open_24h = float(res.result_rows[0][0])
            last_price = float(res.result_rows[0][1])
            summary["change_24h_pct"] = ((last_price - open_24h) / open_24h) * 100
            summary["volume_24h"] = float(res.result_rows[0][2])
            summary["last_price"] = last_price
            
        return summary
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
