package okx

import (
	"context"
	"sync"
	"time"
)

type rule struct {
	n   int
	per time.Duration
}

func every2s(n int) rule { return rule{n, 2 * time.Second} }
func every1s(n int) rule { return rule{n, time.Second} }

// limits mirrors the documented OKX limits per endpoint. Trade endpoints are
// limited per instrument, the rest per IP or user.
var limits = map[string]rule{
	"GET /api/v5/market/tickers":                                every2s(20),
	"GET /api/v5/market/ticker":                                 every2s(20),
	"GET /api/v5/market/index-tickers":                          every2s(20),
	"GET /api/v5/market/books":                                  every2s(40),
	"GET /api/v5/market/books-full":                             every2s(10),
	"GET /api/v5/market/candles":                                every2s(40),
	"GET /api/v5/market/history-candles":                        every2s(20),
	"GET /api/v5/market/index-candles":                          every2s(20),
	"GET /api/v5/market/history-index-candles":                  every2s(10),
	"GET /api/v5/market/mark-price-candles":                     every2s(20),
	"GET /api/v5/market/history-mark-price-candles":             every2s(10),
	"GET /api/v5/market/trades":                                 every2s(100),
	"GET /api/v5/market/history-trades":                         every2s(20),
	"GET /api/v5/market/platform-24-volume":                     every2s(2),
	"GET /api/v5/market/call-auction-details":                   every2s(20),
	"GET /api/v5/public/instruments":                            every2s(20),
	"GET /api/v5/public/time":                                   every2s(10),
	"GET /api/v5/public/funding-rate":                           every2s(10),
	"GET /api/v5/public/funding-rate-history":                   every2s(10),
	"GET /api/v5/public/open-interest":                          every2s(20),
	"GET /api/v5/public/mark-price":                             every2s(10),
	"GET /api/v5/public/price-limit":                            every2s(20),
	"GET /api/v5/public/estimated-price":                        every2s(10),
	"GET /api/v5/public/liquidation-orders":                     every2s(40),
	"GET /api/v5/public/position-tiers":                         every2s(10),
	"GET /api/v5/public/convert-contract-coin":                  every2s(10),
	"GET /api/v5/public/opt-summary":                            every2s(20),
	"GET /api/v5/system/status":                                 {1, 5 * time.Second},
	"GET /api/v5/rubik/stat/taker-volume":                       every2s(5),
	"GET /api/v5/rubik/stat/contracts/long-short-account-ratio": every2s(5),
	"GET /api/v5/account/balance":                               every2s(10),
	"GET /api/v5/account/positions":                             every2s(10),
	"GET /api/v5/account/positions-history":                     every2s(10),
	"GET /api/v5/account/config":                                every2s(5),
	"POST /api/v5/account/set-leverage":                         every2s(20),
	"GET /api/v5/account/leverage-info":                         every2s(20),
	"POST /api/v5/account/set-position-mode":                    every2s(5),
	"GET /api/v5/account/max-size":                              every2s(20),
	"GET /api/v5/account/max-avail-size":                        every2s(20),
	"GET /api/v5/account/trade-fee":                             every2s(5),
	"GET /api/v5/account/bills":                                 every1s(5),
	"GET /api/v5/account/bills-archive":                         every2s(5),
	"POST /api/v5/account/position/margin-balance":              every2s(20),
	"GET /api/v5/account/max-withdrawal":                        every2s(20),
	"POST /api/v5/trade/order":                                  every2s(60),
	"POST /api/v5/trade/batch-orders":                           every2s(300),
	"POST /api/v5/trade/cancel-order":                           every2s(60),
	"POST /api/v5/trade/cancel-batch-orders":                    every2s(300),
	"POST /api/v5/trade/amend-order":                            every2s(60),
	"POST /api/v5/trade/amend-batch-orders":                     every2s(300),
	"POST /api/v5/trade/close-position":                         every2s(20),
	"GET /api/v5/trade/order":                                   every2s(60),
	"GET /api/v5/trade/orders-pending":                          every2s(60),
	"GET /api/v5/trade/orders-history":                          every2s(40),
	"GET /api/v5/trade/orders-history-archive":                  every2s(20),
	"GET /api/v5/trade/fills":                                   every2s(60),
	"GET /api/v5/trade/fills-history":                           every2s(10),
	"POST /api/v5/trade/order-algo":                             every2s(20),
	"POST /api/v5/trade/cancel-algos":                           every2s(20),
	"POST /api/v5/trade/amend-algos":                            every2s(20),
	"GET /api/v5/trade/order-algo":                              every2s(20),
	"GET /api/v5/trade/orders-algo-pending":                     every2s(20),
	"GET /api/v5/trade/orders-algo-history":                     every2s(20),
	"POST /api/v5/trade/cancel-all-after":                       every1s(1),
	"GET /api/v5/asset/currencies":                              every1s(6),
	"GET /api/v5/asset/balances":                                every1s(6),
	"POST /api/v5/asset/transfer":                               every1s(2),
	"GET /api/v5/asset/transfer-state":                          every1s(10),
	"GET /api/v5/asset/deposit-address":                         every1s(6),
	"GET /api/v5/asset/deposit-history":                         every1s(6),
	"POST /api/v5/asset/withdrawal":                             every1s(6),
	"POST /api/v5/asset/cancel-withdrawal":                      every1s(6),
	"GET /api/v5/asset/withdrawal-history":                      every1s(6),
	"GET /api/v5/asset/bills":                                   every1s(6),
}

type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

func newLimiter() *limiter { return &limiter{buckets: map[string]*bucket{}} }

func (l *limiter) wait(ctx context.Context, endpoint, scope string) error {
	r, ok := limits[endpoint]
	if !ok {
		return nil
	}
	key := endpoint + " " + scope
	l.mu.Lock()
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: float64(r.n), capacity: float64(r.n), rate: float64(r.n) / r.per.Seconds(), last: time.Now()}
		l.buckets[key] = b
	}
	l.mu.Unlock()
	return b.wait(ctx)
}

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	last     time.Time
}

func (b *bucket) wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		b.tokens = min(b.capacity, b.tokens+now.Sub(b.last).Seconds()*b.rate)
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		delay := time.Duration((1 - b.tokens) / b.rate * float64(time.Second))
		b.mu.Unlock()

		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}
