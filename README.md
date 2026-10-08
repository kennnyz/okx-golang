# okx-golang

[![Go Reference](https://pkg.go.dev/badge/github.com/kennnyz/okx-golang.svg)](https://pkg.go.dev/github.com/kennnyz/okx-golang)
[![CI](https://github.com/kennnyz/okx-golang/actions/workflows/ci.yml/badge.svg)](https://github.com/kennnyz/okx-golang/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kennnyz/okx-golang)](https://goreportcard.com/report/github.com/kennnyz/okx-golang)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A typed, production-grade Go client for the [OKX v5 API](https://www.okx.com/docs-v5/en/): REST, WebSocket streams and WebSocket trading.

Use it to **automate your trading**, build bots and dashboards, or give **AI agents** safe, typed access to an exchange: every request and response is a Go struct, every error can be matched, and demo trading is one option away.

```go
client := okx.NewClient()
ticker, _ := client.Market.Ticker(ctx, "BTC-USDT")
fmt.Println(ticker.Last) // 82763.2
```

## Why this client

- **Current with the API as of October 2026**: the new `openapi.okx.com` REST domain, WebSocket on port 443 (port 8443 is retired on 31 Oct 2026), `seqId`-based order books instead of the deprecated checksum, `rpi` order types, regional EEA and US domains.
- **Typed WebSocket channels.** `stream.Tickers(...)` gives you a `<-chan Ticker`, not raw bytes. The same goes for order books, candles, orders, positions, balances and more.
- **Connections that heal themselves.** Pings, reconnects with backoff, re-login and re-subscription happen automatically; a hook tells you when to reconcile.
- **WebSocket trading.** Place, amend and cancel orders over the private WebSocket for lower latency.
- **Local order book** with sequence-gap detection and one-call resync.
- **Errors that make sense.** A rejected order returns the real reason (`51008 Insufficient balance`), not "All operations failed". Match it with `errors.Is(err, okx.ErrInsufficientBalance)`.
- **Safe retries.** Rate-limited and expired-timestamp requests are retried because OKX did not process them. Orders are never retried after a network error, so a retry cannot duplicate them.
- **Client-side rate limiter** set to the documented limit of every wrapped endpoint, per instrument where OKX limits per instrument.
- **No precision loss.** Prices and sizes are `okx.Number` (the exact decimal string OKX sent); timestamps are `okx.Time` (a `time.Time`).
- **Pagination as iterators** (Go 1.23 range-over-func): `for fill, err := range client.Trade.AllFills(ctx, req)`.
- **Any endpoint** through `client.Do`, with signing, rate limiting and error handling included.
- **Minimal dependencies**: the standard library plus [`coder/websocket`](https://github.com/coder/websocket).

## Install

```bash
go get github.com/kennnyz/okx-golang
```

Requires Go 1.23 or newer. The package name is `okx`.

## Quick start

Public market data needs no API key:

```go
package main

import (
	"context"
	"fmt"
	"log"

	okx "github.com/kennnyz/okx-golang"
)

func main() {
	ctx := context.Background()
	client := okx.NewClient()

	candles, err := client.Market.Candles(ctx, okx.CandlesRequest{
		InstID: "BTC-USDT",
		Bar:    okx.Bar1H,
		Limit:  24,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles {
		fmt.Println(c.Ts.Format("15:04"), c.Open, c.High, c.Low, c.Close, c.Vol)
	}
}
```

## Authentication, demo trading and regions

Create an API key in OKX → Profile → API. Private endpoints need the key, secret and passphrase:

```go
client := okx.NewClient(
	okx.WithCredentials(os.Getenv("OKX_API_KEY"), os.Getenv("OKX_SECRET_KEY"), os.Getenv("OKX_PASSPHRASE")),
)
```

**Demo trading** uses virtual funds and a separate API key, created inside OKX demo trading mode:

```go
client := okx.NewClient(okx.WithCredentials(key, secret, pass), okx.WithDemoTrading())
```

Accounts registered with **OKX EEA** or **OKX US** must use their region:

```go
client := okx.NewClient(okx.WithRegion(okx.RegionEEA))
```

| Option | Purpose |
|---|---|
| `WithCredentials(key, secret, passphrase)` | Enable private endpoints and channels |
| `WithDemoTrading()` | Route to demo trading |
| `WithRegion(RegionEEA \| RegionUS)` | Use regional domains |
| `WithRetry(n)` | Retries for safe-to-retry failures (default 3, 0 disables) |
| `WithoutRateLimit()` | Disable the client-side rate limiter |
| `WithRequestTTL(d)` | Have OKX reject trade requests that arrive later than `d` |
| `WithHTTPClient(hc)` | Custom `*http.Client` (default timeout 15s) |
| `WithLogger(*slog.Logger)` | Retry and reconnect events |
| `WithBaseURL(url)`, `WithWebSocketURL(url)` | Override hosts |
| `WithStreamBuffer(n)` | Channel capacity of each subscription (default 256) |
| `WithReconnectHook(fn)` | Called after a WebSocket reconnects |

Signatures depend on your clock. If OKX rejects a request because its timestamp expired, the client syncs with server time and retries. You can also call `client.SyncTime(ctx)` at startup.

## Trading

```go
res, err := client.Trade.PlaceOrder(ctx, okx.PlaceOrderRequest{
	InstID:  "BTC-USDT-SWAP",
	TdMode:  okx.TdCross,
	Side:    okx.SideBuy,
	PosSide: okx.PosSideLong, // long/short position mode only
	OrdType: okx.OrdLimit,
	Px:      "80000",
	Sz:      "1", // contracts for SWAP, base currency for SPOT
	ClOrdID: okx.NewClientOrderID(),
	AttachAlgoOrds: []okx.AttachAlgoOrder{{
		TpTriggerPx: "84000", TpOrdPx: "-1", // -1 = market
		SlTriggerPx: "78000", SlOrdPx: "-1",
	}},
})
switch {
case errors.Is(err, okx.ErrInsufficientBalance):
	// top up or reduce size
case err != nil:
	return err
}
fmt.Println("order", res.OrdID)
```

**Idempotency.** Generate `ClOrdID` before sending. If a request times out, look the order up with `client.Trade.Order(ctx, okx.OrderRequest{InstID: ..., ClOrdID: ...})` before sending it again; OKX also rejects a duplicate `clOrdId` among pending orders (code 51016).

**Batch operations** (`PlaceOrders`, `CancelOrders`, `AmendOrders`, `CancelAlgoOrders`) succeed or fail per item. `err` is non-nil only when the whole request failed, so check each result:

```go
results, err := client.Trade.PlaceOrders(ctx, orders)
if err != nil {
	return err
}
for _, r := range results {
	if err := r.Err(); err != nil {
		log.Printf("order %s rejected: %v", r.ClOrdID, err)
	}
}
```

**Algo orders**: TP/SL, OCO, trigger, trailing stop, TWAP, iceberg and chase.

```go
client.Trade.PlaceAlgoOrder(ctx, okx.PlaceAlgoOrderRequest{
	InstID: "BTC-USDT-SWAP", TdMode: okx.TdCross, Side: okx.SideSell, PosSide: okx.PosSideLong,
	OrdType: okx.AlgoTrailing, Sz: "1", CallbackRatio: "0.02", ReduceOnly: true,
})
```

**Dead man's switch.** A bot should call `CancelAllAfter` periodically. If the bot dies, OKX cancels every pending order when the timer runs out:

```go
for range time.Tick(20 * time.Second) {
	client.Trade.CancelAllAfter(ctx, 60*time.Second)
}
```

## Account and positions

```go
bal, _ := client.Account.Balance(ctx, "USDT")
fmt.Println("equity", bal.TotalEq)

positions, _ := client.Account.Positions(ctx, okx.PositionsRequest{InstType: okx.InstSwap})
for _, p := range positions {
	fmt.Println(p.InstID, p.PosSide, p.Pos, "@", p.AvgPx, "uPnL", p.Upl, "liq", p.LiqPx)
}

cfg, _ := client.Account.Config(ctx)
fmt.Println("key permissions:", cfg.Permissions()) // [read_only trade]
```

## Pagination

History endpoints have `All*` iterators that follow the OKX cursors for you:

```go
for fill, err := range client.Trade.AllFills(ctx, okx.FillsRequest{InstType: okx.InstSpot}) {
	if err != nil {
		return err
	}
	fmt.Println(fill.FillTime, fill.InstID, fill.Side, fill.FillSz, "@", fill.FillPx, "fee", fill.Fee)
}
```

Available: `Market.AllHistoryCandles`, `Trade.AllOrdersHistory`, `Trade.AllFills`, `Account.AllBills`, `Account.AllPositionsHistory`. Break out of the loop when you have enough.

## WebSocket streams

```go
stream := okx.NewStream() // add WithCredentials for private channels
defer stream.Close()

tickers, err := stream.Tickers(ctx, "BTC-USDT")
if err != nil {
	return err
}
for t := range tickers.C {
	fmt.Println(t.Last, t.BidPx, t.AskPx)
}
```

Connections open lazily on the right endpoint (public, private or business). Two subscriptions to the same channel share one server subscription. `Unsubscribe` closes the channel; `stream.Close()` closes all of them.

| Public | Private (credentials) |
|---|---|
| `Tickers`, `Trades`, `AllTrades` | `Account` |
| `Candles`, `MarkPriceCandles`, `IndexCandles` | `Positions` |
| `OrderBook` (`Books`, `Books5`, `BBO`, `BooksL2TBT`, `Books50L2TBT`, `BooksRPI`) | `BalanceAndPosition` |
| `MarkPrice`, `IndexTickers`, `FundingRate`, `OpenInterest`, `PriceLimit` | `Orders`, `AlgoOrders` |
| `Liquidations`, `Instruments`, `OptionSummary` | `LiquidationWarning` |

Any other channel: `stream.Subscribe(ctx, okx.Arg{Channel: "...", InstID: "..."})` delivers raw `okx.Push` messages.

Messages are delivered in order and never dropped. A subscriber that stops reading for longer than its buffer stalls its connection, so read promptly or hand messages off to your own queue.

### Reconnects

On disconnect the stream reconnects with exponential backoff, logs in again and restores every subscription. Messages sent while it was disconnected are lost, so reconcile private state when that happens:

```go
stream := okx.NewStream(
	okx.WithCredentials(key, secret, pass),
	okx.WithReconnectHook(func(url string) {
		orders, _ := client.Trade.PendingOrders(ctx, okx.PendingOrdersRequest{})
		syncLocalOrders(orders)
	}),
)
```

### Local order book

```go
sub, _ := stream.OrderBook(ctx, "BTC-USDT", okx.Books)
var book okx.Book
for update := range sub.C {
	if errors.Is(book.Apply(update), okx.ErrBookOutOfSync) {
		sub.Resync(ctx) // request a fresh snapshot
		continue
	}
	if book.Synced() {
		bid, _ := book.BestBid()
		ask, _ := book.BestAsk()
		fmt.Println(bid.Px, ask.Px)
	}
}
```

`Book` checks that every update's `prevSeqId` continues the previous `seqId`, which is what OKX recommends now that checksums are deprecated.

### WebSocket trading

The same requests as REST, sent over the private WebSocket:

```go
res, err := stream.PlaceOrder(ctx, okx.PlaceOrderRequest{...})
stream.AmendOrder(ctx, okx.AmendOrderRequest{InstID: "BTC-USDT", OrdID: res.OrdID, NewPx: "80100"})
stream.CancelOrder(ctx, okx.CancelOrderRequest{InstID: "BTC-USDT", OrdID: res.OrdID})
```

## Errors

Every error from OKX is an `*okx.APIError` with the top-level `Code` and `Msg`, the HTTP status, and per-item codes in `Items`.

```go
var apiErr *okx.APIError
if errors.As(err, &apiErr) {
	log.Println(apiErr.Code, apiErr.Msg, apiErr.Items)
}
if okx.HasCode(err, "51121") { // any OKX code, top level or per item
	// order size is not a multiple of the lot size
}
```

| Sentinel | Meaning |
|---|---|
| `ErrRateLimited` | 50011, 50061, HTTP 429 |
| `ErrAuth` | Invalid key, signature or passphrase, IP not whitelisted, HTTP 401 |
| `ErrTimestampExpired` | Local clock is off (handled automatically) |
| `ErrInvalidParameter` | 50014, 51000, 51001 |
| `ErrInsufficientBalance` | 51008, 51131 |
| `ErrOrderNotFound` | 51603 |
| `ErrServiceUnavailable` | System busy, maintenance, HTTP 502/503/504 |
| `ErrNoCredentials` | Private call without `WithCredentials` |

## Rate limits and retries

Each wrapped endpoint has a token bucket set to its documented OKX limit, and trade endpoints are limited per instrument, as OKX does. Requests wait for a token instead of being rejected. Turn it off with `WithoutRateLimit()` if you run your own limiter.

Retries use exponential backoff with jitter:

| Failure | GET | POST |
|---|---|---|
| Rate limited (OKX did not process it) | retried | retried |
| Expired timestamp | clock synced, retried | clock synced, retried |
| Network error, 5xx, system busy | retried | **not retried** (it may have been processed) |

## Endpoints not wrapped yet

`client.Do` calls any v5 endpoint with signing, rate limiting, retries and error handling:

```go
var subPositions []struct {
	SubPosID string     `json:"subPosId"`
	InstID   string     `json:"instId"`
	Pos      okx.Number `json:"subPos"`
}
err := client.Do(ctx, http.MethodGet, "/api/v5/copytrading/current-subpositions",
	url.Values{"instType": {"SWAP"}}, &subPositions)
```

For GET, params can be `url.Values`, `map[string]string` or a struct with `json` tags; for POST, params is the JSON body.

## Using with AI agents

The client is a solid base for LLM agents that read markets or trade. Every operation is a typed function with a typed result, which maps directly onto tool and function-calling schemas, and every failure has a stable code the agent can reason about.

[docs/ai-agents.md](docs/ai-agents.md) shows how to expose the client as agent tools and which guardrails are worth having: demo mode, read-only keys, `clOrdId` idempotency, request TTL and the dead man's switch. [`llms.txt`](llms.txt) is a compact API reference to put into a model's context.

## Coverage

| Area | Endpoints |
|---|---|
| Market | tickers, ticker, index tickers, order book, full order book, candles, history candles, index and mark price candles, trades, history trades, platform volume, call auction |
| Public | instruments, server time, funding rate and history, open interest, mark price, price limit, estimated price, liquidations, position tiers, contract/coin conversion, option summary, system status, taker volume, long/short ratio |
| Account | balance, positions, positions history, config, set and get leverage, position mode, max order size, max available size, trade fee, bills (7 days and archive), adjust margin, max withdrawal |
| Trade | place, batch place, cancel, batch cancel, amend, batch amend, close position, order, pending orders, history (7 days and archive), fills (3 days and history), cancel all after |
| Algo | place, cancel, amend, get, pending, history |
| Asset | currencies, funding balances, transfer, transfer state, deposit addresses, deposit history, withdraw, cancel withdrawal, withdrawal history, bills |
| WebSocket | 21 typed channel methods, raw subscriptions, place/amend/cancel orders (single and batch) |

Missing something? Use `client.Do` or `stream.Subscribe`, and open an issue or a pull request.

## Examples

| Example | |
|---|---|
| [market-data](examples/market-data) | Ticker, funding rate, 30 days of candles through an iterator |
| [orderbook](examples/orderbook) | Live 400-level local order book with resync |
| [demo-trading](examples/demo-trading) | Limit order with TP/SL on demo trading, dead man's switch |
| [private-stream](examples/private-stream) | Orders and positions stream, reconciled after reconnects |

```bash
go run ./examples/orderbook
```

## Testing

```bash
go test -race ./...                              # unit tests, no network
go test -tags integration -run TestLive ./...    # live public REST and WebSocket
OKX_DEMO_API_KEY=... OKX_DEMO_SECRET_KEY=... OKX_DEMO_PASSPHRASE=... \
  go test -tags integration -run TestLiveDemoPrivate ./...  # demo account: places and cancels an order
```

## Disclaimer

This is an unofficial client and is not affiliated with OKX. Trading crypto assets carries a high risk of loss. Test strategies on demo trading first, and use API keys with the narrowest permissions and an IP whitelist.

## License

[Apache 2.0](LICENSE)
