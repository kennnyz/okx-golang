// Exposes OKX as tools for an LLM agent, with the guardrails enforced in code
// rather than in the prompt.
//
//	go run ./examples/agent-tools                      # print tool definitions
//	go run ./examples/agent-tools get_ticker '{"inst_id":"BTC-USDT"}'
//
// Pass the definitions to any function-calling API or MCP server and route
// the model's tool calls to Toolbox.Call.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	okx "github.com/kennnyz/okx-golang"
)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	handler     func(ctx context.Context, input json.RawMessage) (any, error)
}

type Limits struct {
	Instruments   []string
	MaxOrderQuote float64
	AllowedSides  []okx.Side
}

type Toolbox struct {
	client *okx.Client
	limits Limits
	tools  []Tool
}

func NewToolbox(client *okx.Client, limits Limits) *Toolbox {
	t := &Toolbox{client: client, limits: limits}
	t.tools = []Tool{
		{
			Name:        "get_ticker",
			Description: "Latest price, best bid/ask and 24h stats of an instrument such as BTC-USDT.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"inst_id":{"type":"string"}},"required":["inst_id"]}`),
			handler:     t.getTicker,
		},
		{
			Name:        "get_candles",
			Description: "Recent OHLCV candles, newest first. bar is one of 1m,5m,15m,1H,4H,1D.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"inst_id":{"type":"string"},"bar":{"type":"string"},"limit":{"type":"integer","maximum":300}},"required":["inst_id","bar"]}`),
			handler:     t.getCandles,
		},
		{
			Name:        "get_balance",
			Description: "Trading account equity and per-currency available balance.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			handler:     t.getBalance,
		},
		{
			Name:        "place_limit_order",
			Description: "Place a spot limit order. size is in base currency. Returns the order id.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"inst_id":{"type":"string"},"side":{"type":"string","enum":["buy","sell"]},"price":{"type":"string"},"size":{"type":"string"}},"required":["inst_id","side","price","size"]}`),
			handler:     t.placeLimitOrder,
		},
		{
			Name:        "cancel_order",
			Description: "Cancel a pending order by id.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"inst_id":{"type":"string"},"order_id":{"type":"string"}},"required":["inst_id","order_id"]}`),
			handler:     t.cancelOrder,
		},
	}
	return t
}

func (t *Toolbox) Definitions() []Tool { return t.tools }

// Call runs a tool and always returns JSON for the model: the result, or an
// error with the OKX code so the model can react to it.
func (t *Toolbox) Call(ctx context.Context, name string, input json.RawMessage) json.RawMessage {
	i := slices.IndexFunc(t.tools, func(tool Tool) bool { return tool.Name == name })
	if i < 0 {
		return toolError(fmt.Errorf("unknown tool %q", name))
	}
	result, err := t.tools[i].handler(ctx, input)
	if err != nil {
		return toolError(err)
	}
	out, _ := json.Marshal(map[string]any{"result": result})
	return out
}

func toolError(err error) json.RawMessage {
	body := map[string]any{"error": err.Error()}
	var apiErr *okx.APIError
	if errors.As(err, &apiErr) {
		body["okx_code"] = apiErr.Code
		if len(apiErr.Items) > 0 {
			body["okx_code"] = apiErr.Items[0].Code
		}
	}
	out, _ := json.Marshal(body)
	return out
}

func (t *Toolbox) getTicker(ctx context.Context, input json.RawMessage) (any, error) {
	var in struct {
		InstID string `json:"inst_id"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}
	return t.client.Market.Ticker(ctx, in.InstID)
}

func (t *Toolbox) getCandles(ctx context.Context, input json.RawMessage) (any, error) {
	var in struct {
		InstID string  `json:"inst_id"`
		Bar    okx.Bar `json:"bar"`
		Limit  int     `json:"limit"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}
	return t.client.Market.Candles(ctx, okx.CandlesRequest{InstID: in.InstID, Bar: in.Bar, Limit: min(max(in.Limit, 1), 300)})
}

func (t *Toolbox) getBalance(ctx context.Context, _ json.RawMessage) (any, error) {
	return t.client.Account.Balance(ctx)
}

func (t *Toolbox) placeLimitOrder(ctx context.Context, input json.RawMessage) (any, error) {
	var in struct {
		InstID string     `json:"inst_id"`
		Side   okx.Side   `json:"side"`
		Price  okx.Number `json:"price"`
		Size   okx.Number `json:"size"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}
	if !slices.Contains(t.limits.Instruments, in.InstID) {
		return nil, fmt.Errorf("instrument %s is not allowed; allowed: %v", in.InstID, t.limits.Instruments)
	}
	if !slices.Contains(t.limits.AllowedSides, in.Side) {
		return nil, fmt.Errorf("side %s is not allowed", in.Side)
	}
	if notional := in.Price.Float64() * in.Size.Float64(); notional <= 0 || notional > t.limits.MaxOrderQuote {
		return nil, fmt.Errorf("order value %.2f is outside (0, %.2f]", notional, t.limits.MaxOrderQuote)
	}
	return t.client.Trade.PlaceOrder(ctx, okx.PlaceOrderRequest{
		InstID:  in.InstID,
		TdMode:  okx.TdCash,
		Side:    in.Side,
		OrdType: okx.OrdLimit,
		Px:      in.Price,
		Sz:      in.Size,
		ClOrdID: okx.NewClientOrderID(),
		Tag:     "agent",
	})
}

func (t *Toolbox) cancelOrder(ctx context.Context, input json.RawMessage) (any, error) {
	var in struct {
		InstID  string `json:"inst_id"`
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}
	return t.client.Trade.CancelOrder(ctx, okx.CancelOrderRequest{InstID: in.InstID, OrdID: in.OrderID})
}

// checkKey refuses keys that can withdraw: an agent never needs that.
func checkKey(ctx context.Context, client *okx.Client) error {
	cfg, err := client.Account.Config(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(cfg.Permissions(), "withdraw") {
		return errors.New("API key has withdraw permission; create a key with read and trade only")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := []okx.Option{okx.WithDemoTrading(), okx.WithRequestTTL(5 * time.Second)}
	if key := os.Getenv("OKX_API_KEY"); key != "" {
		opts = append(opts, okx.WithCredentials(key, os.Getenv("OKX_SECRET_KEY"), os.Getenv("OKX_PASSPHRASE")))
	}
	client := okx.NewClient(opts...)
	if os.Getenv("OKX_API_KEY") != "" {
		if err := checkKey(ctx, client); err != nil {
			return err
		}
	}

	tools := NewToolbox(client, Limits{
		Instruments:   []string{"BTC-USDT", "ETH-USDT"},
		MaxOrderQuote: 100,
		AllowedSides:  []okx.Side{okx.SideBuy, okx.SideSell},
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if len(os.Args) < 2 {
		return enc.Encode(tools.Definitions())
	}
	input := json.RawMessage("{}")
	if len(os.Args) > 2 {
		input = json.RawMessage(os.Args[2])
	}
	return enc.Encode(tools.Call(ctx, os.Args[1], input))
}
