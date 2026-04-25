from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import clickhouse_connect
from datetime import datetime
import os

app = FastAPI(title="Cryptalize Brain", description="Streaming Analytics API")

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

class Trade(BaseModel):
    event_id: str
    exchange: str
    pair: str
    price: float
    quantity: float
    side: str
    timestamp_us: datetime

@app.get("/")
async def root():
    return {"message": "Cryptalize Brain is active"}

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
    # Query the latest state of the order book using the ReplacingMergeTree logic
    query = """
        SELECT side, price, quantity
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
