package okx

import "context"

// Tickers streams the ticker of an instrument, pushed up to every 100ms.
func (s *Stream) Tickers(ctx context.Context, instID string) (*Subscription[Ticker], error) {
	return subscribe(ctx, s, Arg{Channel: "tickers", InstID: instID}, decodeData[Ticker])
}

// Trades streams public trades. Trades executed at the same price and side
// in one matching cycle are aggregated, with Count set.
func (s *Stream) Trades(ctx context.Context, instID string) (*Subscription[Trade], error) {
	return subscribe(ctx, s, Arg{Channel: "trades", InstID: instID}, decodeData[Trade])
}

// AllTrades streams every individual public trade.
func (s *Stream) AllTrades(ctx context.Context, instID string) (*Subscription[Trade], error) {
	return subscribe(ctx, s, Arg{Channel: "trades-all", InstID: instID}, decodeData[Trade])
}

// Candles streams the current candle of a bar size, pushed up to every
// 500ms. A candle with Confirmed set is closed.
func (s *Stream) Candles(ctx context.Context, instID string, bar Bar) (*Subscription[Candle], error) {
	return subscribe(ctx, s, Arg{Channel: "candle" + string(bar), InstID: instID}, decodeData[Candle])
}

// MarkPriceCandles streams mark price candles.
func (s *Stream) MarkPriceCandles(ctx context.Context, instID string, bar Bar) (*Subscription[Candle], error) {
	return subscribe(ctx, s, Arg{Channel: "mark-price-candle" + string(bar), InstID: instID}, decodeData[Candle])
}

// IndexCandles streams index price candles of an index such as "BTC-USD".
func (s *Stream) IndexCandles(ctx context.Context, instID string, bar Bar) (*Subscription[Candle], error) {
	return subscribe(ctx, s, Arg{Channel: "index-candle" + string(bar), InstID: instID}, decodeData[Candle])
}

// BookChannel selects the depth and speed of an order book channel.
type BookChannel string

const (
	// Books is 400 levels: a snapshot, then incremental updates every 100ms.
	Books BookChannel = "books"
	// Books5 is a full 5-level snapshot every 100ms.
	Books5 BookChannel = "books5"
	// BBO is the best bid and ask, pushed every 10ms on change.
	BBO BookChannel = "bbo-tbt"
	// BooksL2TBT is 400 levels updated every 10ms; requires VIP5+.
	BooksL2TBT BookChannel = "books-l2-tbt"
	// Books50L2TBT is 50 levels updated every 10ms; requires VIP4+.
	Books50L2TBT BookChannel = "books50-l2-tbt"
	// BooksRPI is the book including retail price improvement orders.
	BooksRPI BookChannel = "books-rpi"
)

// BookUpdate is one order book push. Action is "snapshot" or "update"; it
// is empty for channels that always push full snapshots.
type BookUpdate struct {
	InstID string
	Action string
	OrderBook
}

// IsSnapshot reports whether the update replaces the whole book.
func (u BookUpdate) IsSnapshot() bool { return u.Action != "update" }

// OrderBook streams order book changes. Feed incremental channels into a
// Book to maintain the full depth.
func (s *Stream) OrderBook(ctx context.Context, instID string, channel BookChannel) (*Subscription[BookUpdate], error) {
	return subscribe(ctx, s, Arg{Channel: string(channel), InstID: instID}, func(p Push) ([]BookUpdate, error) {
		books, err := decodeData[OrderBook](p)
		if err != nil {
			return nil, err
		}
		out := make([]BookUpdate, len(books))
		for i, b := range books {
			out[i] = BookUpdate{InstID: p.Arg.InstID, Action: p.Action, OrderBook: b}
		}
		return out, nil
	})
}

// MarkPrice streams the mark price, pushed every 200ms on change.
func (s *Stream) MarkPrice(ctx context.Context, instID string) (*Subscription[MarkPrice], error) {
	return subscribe(ctx, s, Arg{Channel: "mark-price", InstID: instID}, decodeData[MarkPrice])
}

// IndexTickers streams an index such as "BTC-USDT".
func (s *Stream) IndexTickers(ctx context.Context, instID string) (*Subscription[IndexTicker], error) {
	return subscribe(ctx, s, Arg{Channel: "index-tickers", InstID: instID}, decodeData[IndexTicker])
}

// FundingRate streams the funding rate of a perpetual swap.
func (s *Stream) FundingRate(ctx context.Context, instID string) (*Subscription[FundingRate], error) {
	return subscribe(ctx, s, Arg{Channel: "funding-rate", InstID: instID}, decodeData[FundingRate])
}

// OpenInterest streams open interest, pushed every 3s.
func (s *Stream) OpenInterest(ctx context.Context, instID string) (*Subscription[OpenInterest], error) {
	return subscribe(ctx, s, Arg{Channel: "open-interest", InstID: instID}, decodeData[OpenInterest])
}

// PriceLimit streams the price band of an instrument.
func (s *Stream) PriceLimit(ctx context.Context, instID string) (*Subscription[PriceLimit], error) {
	return subscribe(ctx, s, Arg{Channel: "price-limit", InstID: instID}, decodeData[PriceLimit])
}

// Liquidations streams liquidations of all instruments of a type, at most
// one per instrument per second.
func (s *Stream) Liquidations(ctx context.Context, instType InstType) (*Subscription[Liquidation], error) {
	return subscribe(ctx, s, Arg{Channel: "liquidation-orders", InstType: instType}, decodeData[Liquidation])
}

// Instruments streams instrument listings and changes.
func (s *Stream) Instruments(ctx context.Context, instType InstType) (*Subscription[Instrument], error) {
	return subscribe(ctx, s, Arg{Channel: "instruments", InstType: instType}, decodeData[Instrument])
}

// OptionSummary streams greeks of all options of a family such as "BTC-USD".
func (s *Stream) OptionSummary(ctx context.Context, instFamily string) (*Subscription[OptionSummary], error) {
	return subscribe(ctx, s, Arg{Channel: "opt-summary", InstFamily: instFamily}, decodeData[OptionSummary])
}

// Account streams the trading account balance on every change. An empty
// ccy streams all currencies. Requires credentials.
func (s *Stream) Account(ctx context.Context, ccy string) (*Subscription[Balance], error) {
	return subscribe(ctx, s, Arg{Channel: "account", Ccy: ccy}, decodeData[Balance])
}

// Positions streams open positions. Use InstAny for every instrument type and
// an empty instID for every instrument. Requires credentials.
func (s *Stream) Positions(ctx context.Context, instType InstType, instID string) (*Subscription[Position], error) {
	return subscribe(ctx, s, Arg{Channel: "positions", InstType: instType, InstID: instID}, decodeData[Position])
}

// Orders streams order updates: placement, fills, amendments and
// cancellations. Use InstAny for every instrument type. Requires credentials.
func (s *Stream) Orders(ctx context.Context, instType InstType, instID string) (*Subscription[Order], error) {
	return subscribe(ctx, s, Arg{Channel: "orders", InstType: instType, InstID: instID}, decodeData[Order])
}

// AlgoOrders streams algo order updates. Requires credentials.
func (s *Stream) AlgoOrders(ctx context.Context, instType InstType, instID string) (*Subscription[AlgoOrder], error) {
	return subscribe(ctx, s, Arg{Channel: "orders-algo", InstType: instType, InstID: instID}, decodeData[AlgoOrder])
}

// LiquidationWarning streams positions approaching liquidation. Requires
// credentials.
func (s *Stream) LiquidationWarning(ctx context.Context, instType InstType) (*Subscription[Position], error) {
	return subscribe(ctx, s, Arg{Channel: "liquidation-warning", InstType: instType}, decodeData[Position])
}

// BalanceAndPosition is a combined balance and position change caused by
// one event such as a fill, funding fee or liquidation.
type BalanceAndPosition struct {
	EventType string             `json:"eventType"`
	PTime     Time               `json:"pTime"`
	BalData   []BalanceSnapshot  `json:"balData"`
	PosData   []PositionSnapshot `json:"posData"`
	Trades    []struct {
		InstID  string `json:"instId"`
		TradeID string `json:"tradeId"`
	} `json:"trades"`
}

type BalanceSnapshot struct {
	Ccy     string `json:"ccy"`
	CashBal Number `json:"cashBal"`
	UTime   Time   `json:"uTime"`
}

type PositionSnapshot struct {
	PosID          string   `json:"posId"`
	TradeID        string   `json:"tradeId"`
	InstID         string   `json:"instId"`
	InstType       InstType `json:"instType"`
	MgnMode        MgnMode  `json:"mgnMode"`
	PosSide        PosSide  `json:"posSide"`
	Pos            Number   `json:"pos"`
	Ccy            string   `json:"ccy"`
	PosCcy         string   `json:"posCcy"`
	AvgPx          Number   `json:"avgPx"`
	NonSettleAvgPx Number   `json:"nonSettleAvgPx"`
	SettledPnl     Number   `json:"settledPnl"`
	UTime          Time     `json:"uTime"`
}

// BalanceAndPosition streams balance and position changes together, so
// both sides of a fill arrive in one message. Requires credentials.
func (s *Stream) BalanceAndPosition(ctx context.Context) (*Subscription[BalanceAndPosition], error) {
	return subscribe(ctx, s, Arg{Channel: "balance_and_position"}, decodeData[BalanceAndPosition])
}
