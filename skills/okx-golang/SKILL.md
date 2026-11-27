---
name: okx-golang
description: Write Go code for the OKX crypto exchange with the okx-golang SDK (github.com/kennnyz/okx-golang). Use when building an OKX trading bot, market data collector, order or position tracker, WebSocket feed, local order book, or AI agent tools for OKX in Go, or when Go code calls the OKX v5 API.
---

# OKX in Go with okx-golang

`github.com/kennnyz/okx-golang` (package `okx`, Go 1.23+) is a typed SDK for the OKX v5 API: REST, typed WebSocket channels, WebSocket trading and a local order book. It signs requests, rate-limits per endpoint, retries only what is safe to retry, and reconnects streams on its own. Do not hand-roll HMAC signing or WebSocket handling for OKX: use this package.

```bash
go get github.com/kennnyz/okx-golang
```

The full API reference is in https://raw.githubusercontent.com/kennnyz/okx-golang/main/llms.txt. Fetch it when you need a method or field this file does not show.

## Setup

```go
import okx "github.com/kennnyz/okx-golang"

client := okx.NewClient() // public data, no key needed

client := okx.NewClient(
	okx.WithCredentials(os.Getenv("OKX_API_KEY"), os.Getenv("OKX_SECRET_KEY"), os.Getenv("OKX_PASSPHRASE")),
	okx.WithDemoTrading(), // virtual funds; needs a key created in OKX demo trading mode
)
```

- Read keys from the environment. Never hard-code or log them.
- Start every new bot on `WithDemoTrading()`. Remove it only when the user asks to go live.
- Accounts registered with OKX EEA or OKX US need `okx.WithRegion(okx.RegionEEA)` or `okx.WithRegion(okx.RegionUS)`.
- `NewClient` and `NewStream` take the same options. Both are safe for concurrent use.

## Market data

```go
ticker, err := client.Market.Ticker(ctx, "BTC-USDT")
fmt.Println(ticker.Last, ticker.BidPx, ticker.AskPx)

candles, err := client.Market.Candles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar1H, Limit: 100})
// newest first; c.Confirmed is false for the candle still forming

for c, err := range client.Market.AllHistoryCandles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar1D}) {
	if err != nil {
		return err
	}
	// walks back through history; break when you have enough
}
```

Instrument IDs: spot `BTC-USDT`, perpetual `BTC-USDT-SWAP`, futures `BTC-USDT-261225`, index `BTC-USD`.

## Orders

```go
// Spot market buy for 50 USDT
res, err := client.Trade.PlaceOrder(ctx, okx.PlaceOrderRequest{
	InstID:  "BTC-USDT",
	TdMode:  okx.TdCash,
	Side:    okx.SideBuy,
	OrdType: okx.OrdMarket,
	Sz:      "50",
	TgtCcy:  okx.TgtQuoteCcy, // Sz is in USDT; the default for spot market buys
	ClOrdID: okx.NewClientOrderID(),
})

// Spot limit sell with a price
res, err = client.Trade.PlaceOrder(ctx, okx.PlaceOrderRequest{
	InstID: "BTC-USDT", TdMode: okx.TdCash, Side: okx.SideSell,
	OrdType: okx.OrdLimit, Px: "90000", Sz: "0.001", ClOrdID: okx.NewClientOrderID(),
})
```

- `Sz` is base currency for spot and contracts for SWAP/FUTURES.
- Prices and sizes are strings (`okx.Number` in responses). Never put money in `float64`.
- Before placing, round `Px` to `TickSz` and `Sz` to `LotSz`, and check `MinSz`. Get them once from `client.Public.Instruments(ctx, okx.InstrumentsRequest{InstType: okx.InstSpot})`.
- Set `ClOrdID` with `okx.NewClientOrderID()`. After a timeout, look the order up with `client.Trade.Order(ctx, okx.OrderRequest{InstID: ..., ClOrdID: ...})` instead of sending it again. The SDK never resends an order after a network error.
- Batch calls (`PlaceOrders`, `CancelOrders`, `AmendOrders`) succeed or fail per item: check `r.Err()` for every result even when `err == nil`.
- A long-running bot should arm the dead man's switch: call `client.Trade.CancelAllAfter(ctx, 60*time.Second)` every 20 seconds, so OKX cancels pending orders if the bot dies.

## Account

```go
bal, err := client.Account.Balance(ctx, "USDT", "BTC")
for _, d := range bal.Details {
	fmt.Println(d.Ccy, d.AvailBal)
}

for fill, err := range client.Trade.AllFills(ctx, okx.FillsRequest{InstType: okx.InstSpot}) {
	if err != nil {
		return err
	}
	fmt.Println(fill.InstID, fill.Side, fill.FillSz, fill.FillPx, fill.Fee)
}
```

## WebSocket streams

```go
stream := okx.NewStream() // add WithCredentials for private channels
defer stream.Close()

sub, err := stream.Tickers(ctx, "BTC-USDT")
if err != nil {
	return err
}
for t := range sub.C {
	fmt.Println(t.Last)
}
```

- Channels are typed: `Tickers`, `Trades`, `Candles`, `OrderBook`, `FundingRate` and more; private ones are `Account`, `Positions`, `Orders`, `AlgoOrders`.
- Reconnects, re-login and re-subscription are automatic. Messages sent while disconnected are lost, so pass `okx.WithReconnectHook(func(url string) { ... })` and re-read open orders and balances from REST when it fires.
- Local order book: subscribe with `stream.OrderBook(ctx, instID, okx.Books)`, feed each update to `var book okx.Book` with `book.Apply(update)`, and call `sub.Resync(ctx)` when it returns `okx.ErrBookOutOfSync`.
- `stream.PlaceOrder`, `CancelOrder` and `AmendOrder` trade over the WebSocket with the same request types as REST.

## Errors

```go
switch {
case errors.Is(err, okx.ErrInsufficientBalance):
case errors.Is(err, okx.ErrRateLimited):
case okx.HasCode(err, "51121"): // any OKX code: order size is not a multiple of the lot size
case err != nil:
	var apiErr *okx.APIError
	if errors.As(err, &apiErr) {
		log.Println(apiErr.Code, apiErr.Msg, apiErr.Items)
	}
}
```

Rate limits and safe retries are handled inside the SDK. Do not add your own retry loop around order placement.

## Endpoints the SDK does not wrap

```go
var out []struct {
	InstID string     `json:"instId"`
	Pos    okx.Number `json:"subPos"`
}
err := client.Do(ctx, http.MethodGet, "/api/v5/copytrading/current-subpositions", url.Values{"instType": {"SWAP"}}, &out)
```

`client.Do` signs, rate-limits and decodes the `data` field. For an unwrapped WebSocket channel use `stream.Subscribe(ctx, okx.Arg{Channel: "...", InstID: "..."})`.

## AI agent tools

When exposing OKX to an LLM as tools, enforce limits in the tool handler, not in the prompt: a read-only or trade-only key (refuse to start if `cfg.Permissions()` from `client.Account.Config` contains `withdraw`), an instrument allowlist, a maximum order value, `okx.WithRequestTTL(5*time.Second)`, `ClOrdID` generated in code, and the dead man's switch. See https://github.com/kennnyz/okx-golang/blob/main/docs/ai-agents.md and the runnable `examples/agent-tools`.
