# Using okx-golang with AI agents

LLM agents work well with exchanges when the exchange is exposed as a small set of well-described tools and every dangerous decision is checked in code. This guide shows how to do that with okx-golang. A complete, runnable version is in [examples/agent-tools](../examples/agent-tools).

## Why a typed client helps

- **Tools map one-to-one onto methods.** `client.Market.Ticker`, `client.Account.Balance` and `client.Trade.PlaceOrder` each become one tool. The request structs tell you which inputs the tool needs; the response structs serialize to clean JSON for the model.
- **Numbers stay exact.** Prices and sizes are `okx.Number` strings, so values pass from the model to OKX and back without float rounding.
- **Errors are stable codes.** Return `okx_code` to the model (`51008` insufficient balance, `51121` wrong lot size, `50011` rate limit) and it can correct itself instead of guessing from prose.
- **Rate limits and retries are handled below the agent**, so a chatty agent cannot get the key banned and does not need retry logic in its prompt.

## Shape of a tool layer

```go
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	handler     func(ctx context.Context, input json.RawMessage) (any, error)
}
```

1. Describe each tool with a JSON schema and a one-line description written for the model.
2. Decode the model's input into a small struct, validate it, call the client.
3. Return `{"result": ...}` or `{"error": "...", "okx_code": "..."}`.

The same definitions work with any function-calling API (Anthropic, OpenAI, Gemini) and inside an MCP server.

Start with read-only tools: ticker, candles, order book, funding rate, open interest, balance, positions, open orders, fills. Many agents never need more than these. Add trading tools only when you have the guardrails below.

## Guardrails

Enforce limits in code. A prompt that says "never trade more than $100" is a suggestion; a check in the tool handler is a rule.

**Separate keys per job.** An analysis agent gets a read-only key. A trading agent gets read and trade, never withdraw. Check at startup and refuse to run otherwise:

```go
cfg, err := client.Account.Config(ctx)
if slices.Contains(cfg.Permissions(), "withdraw") {
	return errors.New("refusing to run with a withdraw-enabled key")
}
```

Bind every key to an IP whitelist in OKX.

**Start on demo trading.** `okx.WithDemoTrading()` with a demo key runs the same code against virtual funds. Keep the agent there until its behavior is boring.

**Allowlist instruments and cap order value.** Check `price × size` against a maximum before calling `PlaceOrder`, and reject instruments the agent was not meant to touch. Load `Public.Instruments` once to validate `tickSz`, `lotSz` and `minSz` before sending, so the agent gets a precise error.

**Generate `ClOrdID` in your code, not in the model.** `okx.NewClientOrderID()` per order intent. If a call times out, look the order up by `ClOrdID` before placing it again.

**Expire slow requests.** `okx.WithRequestTTL(5 * time.Second)` makes OKX reject an order that reaches the matching engine more than five seconds after it was sent, so a slow agent loop cannot fill at a stale price.

**Arm the dead man's switch.** While the agent runs, call `client.Trade.CancelAllAfter(ctx, 60*time.Second)` every 20 seconds. If the process hangs or crashes, OKX cancels its pending orders.

**Do the math in code.** Let the agent decide intent ("buy 50 USDT of BTC at 81000"); compute the size, round it to the lot size and check limits in the handler. Models are unreliable at decimal arithmetic.

**Confirm large actions with a human.** Above a threshold, return a "needs confirmation" result instead of trading, and resume after approval.

**Log every tool call** with inputs, results and OKX codes. When an agent misbehaves, this log is how you find out why.

## Feeding market context

For an agent that reasons about the market, summarize before you send. Fifty raw candles cost far more tokens than a compact table:

```go
candles, _ := client.Market.Candles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar4H, Limit: 50})
var b strings.Builder
for _, c := range candles {
	fmt.Fprintf(&b, "%s O%s H%s L%s C%s V%s\n", c.Ts.UTC().Format("01-02 15:04"), c.Open, c.High, c.Low, c.Close, c.Vol)
}
```

Funding rate, open interest, taker volume (`Public.TakerVolume`) and the long/short ratio (`Public.LongShortAccountRatio`) are useful signals in a few hundred tokens.

For agents that react to events, run a `Stream` in your program and wake the agent on the events you care about (an order fill, a liquidation warning, a price crossing a level) instead of letting it poll.

## Giving the model the API

[`llms.txt`](../llms.txt) at the repository root is a compact reference of the whole package. Put it into the context of a coding assistant when it writes code against this client.
