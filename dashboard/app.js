// --- Global Utils ---
function updateGlobalTime() {
    const el = document.getElementById('current-time');
    if (el) {
        el.textContent = new Date().toISOString().replace('T', ' ').substring(0, 23) + ' UTC';
    }
}
setInterval(updateGlobalTime, 15);

function esc(s) {
    return String(s).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');
}

// --- Status Page Logic ---
async function initStatusPage() {
    console.log("Initializing Status Page...");
    
    async function refreshStatus() {
        try {
            const res = await fetch('/api/status');
            const services = await res.json();
            renderServices(services);
        } catch (e) {
            console.error("Failed to fetch status:", e);
        }
    }

    function renderServices(services) {
        const container = document.getElementById('services-list');
        const countEl = document.getElementById('services-count');
        const nominalEl = document.getElementById('nominal-count');
        const overallText = document.getElementById('overall-status-text');
        const overallBox = document.getElementById('overall-status-box');
        const mainIcon = document.getElementById('main-status-icon');

        if (!container) return;
        container.innerHTML = '';
        countEl.textContent = services.length;

        let nominal = 0;
        let hasFailure = false;
        let hasDegraded = false;

        services.forEach(s => {
            if (s.status === 'UP') nominal++;
            else if (s.critical) hasFailure = true;
            else hasDegraded = true;

            const statusClass = s.status === 'UP' ? 'operational' : (s.critical ? 'failure' : 'degraded');
            const colorClass = s.status === 'UP' ? 'cg' : (s.critical ? 'cr' : 'cy');
            
            const div = document.createElement('div');
            div.className = `service ${statusClass}`;
            div.innerHTML = `
                <div class="service-icon">
                    <img src="/assets/status-${statusClass}.svg" class="${statusClass !== 'operational' ? 'blink' : ''}">
                </div>
                <div class="service-info">
                    <div class="service-header">
                        <div class="service-title">
                            <h3 class="cy">${esc(s.name)}</h3>
                            <div class="co" style="font-size: 10px;">${esc(s.description)}</div>
                        </div>
                        <div style="text-align: right;">
                            <div class="${colorClass}" style="font-weight: bold; font-size: 18px;">${s.status}</div>
                            <div class="cy" style="font-size: 10px;">UPTIME: <span class="cg">${(s.uptime * 100).toFixed(2)}%</span></div>
                        </div>
                    </div>
                    <div class="service-checks">
                        ${s.checks.map(c => `<div class="sc sc-${c}"></div>`).join('')}
                    </div>
                </div>
            `;
            container.appendChild(div);
        });

        nominalEl.textContent = `${nominal} / ${services.length} NOMINAL`;
        
        const overallStatus = hasFailure ? 'failure' : (hasDegraded ? 'degraded' : 'operational');
        overallBox.className = `system-status ${overallStatus}`;
        overallText.textContent = overallStatus.toUpperCase();
        overallText.className = overallStatus === 'operational' ? 'cg' : (overallStatus === 'failure' ? 'cr' : 'cy');
        mainIcon.src = `/assets/status-${overallStatus}.svg`;
    }

    refreshStatus();
    setInterval(refreshStatus, 15000);
}

// --- Dashboard Logic ---
let chart, candleSeries;
let currentExchange = 'binance';
let currentPair = 'BTCUSDT';
let currentInterval = '1m';

const EXCHANGE_PAIRS = {
    binance: [
        { value: 'BTCUSDT', label: 'BTC/USDT' },
        { value: 'ETHUSDT', label: 'ETH/USDT' }
    ],
    kraken: [
        { value: 'BTC/USD', label: 'BTC/USD' },
        { value: 'ETH/USD', label: 'ETH/USD' }
    ]
};

function updatePairOptions() {
    const pairSelect = document.getElementById('pair-select');
    if (!pairSelect) return;
    
    const pairs = EXCHANGE_PAIRS[currentExchange] || [];
    pairSelect.innerHTML = pairs.map(p => `<option value="${p.value}">${p.label}</option>`).join('');
    
    // Set first pair as current
    if (pairs.length > 0) {
        currentPair = pairs[0].value;
    }
}

async function initDashboard() {
    console.log("Initializing Dashboard...");
    
    // Init Chart
    const chartContainer = document.getElementById('price-chart');
    if (!chartContainer) {
        console.error("Chart container not found");
        return;
    }

    try {
        chart = LightweightCharts.createChart(chartContainer, {
            layout: {
                background: { type: LightweightCharts.ColorType.Solid, color: '#050505' },
                textColor: '#ff873d',
            },
            grid: {
                vertLines: { color: '#111' },
                horzLines: { color: '#111' },
            },
            crosshair: {
                mode: LightweightCharts.CrosshairMode.Normal,
            },
            rightPriceScale: {
                borderColor: '#ff873d',
            },
            timeScale: {
                borderColor: '#ff873d',
                timeVisible: true,
            },
        });

        candleSeries = chart.addCandlestickSeries({
            upColor: '#90ffd4',
            downColor: '#ff0047',
            borderDownColor: '#ff0047',
            borderUpColor: '#90ffd4',
            wickDownColor: '#ff0047',
            wickUpColor: '#90ffd4',
        });
        
        console.log("Chart initialized successfully");
    } catch (e) {
        console.error("Failed to initialize chart:", e);
    }

    // Event Listeners
    updatePairOptions(); // Initial pairs

    document.getElementById('exchange-select').addEventListener('change', e => {
        currentExchange = e.target.value;
        updatePairOptions();
        updateMarket();
    });
    document.getElementById('pair-select').addEventListener('change', e => {
        currentPair = e.target.value;
        updateMarket();
    });

    document.querySelectorAll('.interval-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.interval-btn').forEach(b => {
                b.style.borderColor = 'var(--orange)';
                b.style.color = 'var(--orange)';
                b.className = 'interval-btn';
            });
            btn.style.borderColor = 'var(--yellow)';
            btn.style.color = 'var(--yellow)';
            btn.className = 'interval-btn cy';
            currentInterval = btn.dataset.interval;
            fetchChartData();
        });
    });

    // Data Loop
    async function updateMarket() {
        await Promise.all([
            fetchChartData(),
            fetchSummary(),
            fetchTrades(),
            fetchOrderBook(),
            updateTicker()
        ]);
    }

    let globalSummaries = [];

    async function updateTicker() {
        const ticker = document.getElementById('ticker-content');
        if (!ticker) return;

        try {
            const res = await fetch('/analytics/all_summaries');
            if (res.ok) {
                globalSummaries = await res.json();
            }
        } catch (e) { console.error("Ticker fetch failed", e); }

        if (globalSummaries.length === 0) return;

        const messages = globalSummaries.map(s => {
            const price = s.last_price ? s.last_price.toFixed(2) : '---';
            const change = s.change_24h_pct ? s.change_24h_pct.toFixed(2) : '0.00';
            const colorClass = parseFloat(change) >= 0 ? 'cg' : 'cr';
            return `
                <span class="co" style="margin-right: 10px;">[${s.exchange.toUpperCase()}]</span>
                <span class="cy">${s.pair}:</span>
                <span class="cw">${price}</span>
                <span class="${colorClass}" style="margin-right: 40px;">(${change}%)</span>
            `;
        });
        
        // Quadruple for smooth loop
        ticker.innerHTML = messages.join('') + messages.join('') + messages.join('') + messages.join('');
    }

    async function fetchSummary() {
        try {
            const res = await fetch(`/analytics/summary/${currentExchange}/${currentPair}`);
            if (!res.ok) {
                console.error(`Summary fetch failed: ${res.status}`);
                return;
            }
            const data = await res.json();
            if (!data || data.detail) {
                console.warn("Summary data invalid:", data?.detail);
                return;
            }
            
            lastSummaryData = data; // Store for ticker
            
            document.getElementById('last-price').textContent = data.last_price ? data.last_price.toFixed(2) : '---';
            const changeEl = document.getElementById('price-change');
            if (data.change_24h_pct !== undefined) {
                changeEl.textContent = (data.change_24h_pct >= 0 ? '+' : '') + data.change_24h_pct.toFixed(2) + '%';
                changeEl.className = 'stat-value ' + (data.change_24h_pct >= 0 ? 'cg' : 'cr');
            }
            document.getElementById('volume-24h').textContent = data.volume_24h ? data.volume_24h.toFixed(2) : '---';
            document.getElementById('vwap-val').textContent = data.vwap ? data.vwap.toFixed(2) : '---';
            
            if (data.spread) {
                document.getElementById('spread-val').textContent = data.spread.spread.toFixed(2);
                document.getElementById('spread-bps').textContent = data.spread.spread_bps.toFixed(2) + ' BPS';
            }
            
            if (data.imbalance !== undefined) {
                document.getElementById('imbalance-val').textContent = data.imbalance.toFixed(3);
                document.getElementById('imbalance-fill').style.width = (data.imbalance * 100) + '%';
            }
        } catch (e) { console.error("Error fetching summary:", e); }
    }

    async function fetchChartData() {
        try {
            const res = await fetch(`/analytics/ohlcv/${currentExchange}/${currentPair}?interval=${currentInterval}&limit=100`);
            if (!res.ok) {
                console.error(`Chart fetch failed: ${res.status}`);
                return;
            }
            const data = await res.json();
            if (!data || data.length === 0 || data.detail) {
                console.warn("No chart data received:", data?.detail);
                return;
            }
            const candles = data.map(c => ({
                time: Math.floor(new Date(c.bucket).getTime() / 1000),
                open: c.open,
                high: c.high,
                low: c.low,
                close: c.close
            })).sort((a, b) => a.time - b.time);
            
            // Filter duplicates (Lightweight Charts requirement)
            const uniqueCandles = [];
            const seenTimes = new Set();
            for (const c of candles) {
                if (!seenTimes.has(c.time)) {
                    uniqueCandles.push(c);
                    seenTimes.add(c.time);
                }
            }

            if (candleSeries) {
                candleSeries.setData(uniqueCandles);
            }
        } catch (e) { console.error("Error fetching chart data:", e); }
    }

    async function fetchTrades() {
        try {
            const res = await fetch(`/trades/${currentExchange}/${currentPair}?limit=20`);
            if (!res.ok) return;
            const data = await res.json();
            if (data.detail) return;
            
            const list = document.getElementById('trades-list');
            list.innerHTML = data.map(t => `
                <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
                    <td style="padding: 4px;">${new Date(t.timestamp_us).toLocaleTimeString()}</td>
                    <td class="cy">${t.price.toFixed(2)}</td>
                    <td>${t.quantity.toFixed(4)}</td>
                    <td style="text-align: right;" class="${t.side.endsWith('BUY') ? 'cg' : 'cr'}">${t.side.endsWith('BUY') ? 'BUY' : 'SELL'}</td>
                </tr>
            `).join('');
        } catch (e) { console.error("Error fetching trades:", e); }
    }

    async function fetchOrderBook() {
        try {
            const res = await fetch(`/orderbook/${currentExchange}/${currentPair}`);
            if (!res.ok) return;
            const data = await res.json();
            if (data.detail) return;
            
            const bidsList = document.getElementById('bids-list');
            const asksList = document.getElementById('asks-list');
            
            bidsList.innerHTML = data.bids.slice(0, 15).map(b => `
                <div class="ob-row bid">
                    <span>${b.quantity.toFixed(4)}</span>
                    <span>${b.price.toFixed(2)}</span>
                </div>
            `).join('');

            asksList.innerHTML = data.asks.slice(-15).map(a => `
                <div class="ob-row ask">
                    <span>${a.price.toFixed(2)}</span>
                    <span>${a.quantity.toFixed(4)}</span>
                </div>
            `).join('');
        } catch (e) { console.error("Error fetching orderbook:", e); }
    }

    updateMarket();
    setInterval(updateMarket, 2000);
}

window.initStatusPage = initStatusPage;
window.initDashboard = initDashboard;
