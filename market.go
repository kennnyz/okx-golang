package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
)

// MarketService covers /api/v5/market: prices, order books, candles, trades.
type MarketService struct{ c *Client }

type Ticker struct {
	InstType  InstType `json:"instType"`
	InstID    string   `json:"instId"`
	Last      Number   `json:"last"`
	LastSz    Number   `json:"lastSz"`
	AskPx     Number   `json:"askPx"`
	AskSz     Number   `json:"askSz"`
	BidPx     Number   `json:"bidPx"`
	BidSz     Number   `json:"bidSz"`
	Open24h   Number   `json:"open24h"`
	High24h   Number   `json:"high24h"`
	Low24h    Number   `json:"low24h"`
	VolCcy24h Number   `json:"volCcy24h"`
	Vol24h    Number   `json:"vol24h"`
	SodUtc0   Number   `json:"sodUtc0"`
	SodUtc8   Number   `json:"sodUtc8"`
	Ts        Time     `json:"ts"`
}

type TickersRequest struct {
	InstType   InstType `json:"instType"`
	InstFamily string   `json:"instFamily,omitempty"`
}

// Tickers returns the latest ticker of every instrument of one type.
func (s *MarketService) Tickers(ctx context.Context, req TickersRequest) ([]Ticker, error) {
	return list[Ticker](ctx, s.c, publicGet("/api/v5/market/tickers", req))
}

// Ticker returns the latest ticker of one instrument, e.g. "BTC-USDT".
func (s *MarketService) Ticker(ctx context.Context, instID string) (Ticker, error) {
	return one[Ticker](ctx, s.c, publicGet("/api/v5/market/ticker", map[string]string{"instId": instID}))
}

type IndexTicker struct {
	InstID  string `json:"instId"`
	IdxPx   Number `json:"idxPx"`
	High24h Number `json:"high24h"`
	Low24h  Number `json:"low24h"`
	Open24h Number `json:"open24h"`
	SodUtc0 Number `json:"sodUtc0"`
	SodUtc8 Number `json:"sodUtc8"`
	Ts      Time   `json:"ts"`
}

type IndexTickersRequest struct {
	QuoteCcy string `json:"quoteCcy,omitempty"`
	InstID   string `json:"instId,omitempty"`
}

func (s *MarketService) IndexTickers(ctx context.Context, req IndexTickersRequest) ([]IndexTicker, error) {
	return list[IndexTicker](ctx, s.c, publicGet("/api/v5/market/index-tickers", req))
}

// BookLevel is one price level of an order book.
type BookLevel struct {
	Px     Number
	Sz     Number
	Orders int64
}

func (l *BookLevel) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) < 3 {
		return fmt.Errorf("okx: book level has %d fields", len(raw))
	}
	if err := json.Unmarshal(raw[0], &l.Px); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &l.Sz); err != nil {
		return err
	}
	var orders Number
	if err := json.Unmarshal(raw[len(raw)-1], &orders); err != nil {
		return err
	}
	l.Orders = orders.Int64()
	return nil
}

func (l BookLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{string(l.Px), string(l.Sz), "0", fmt.Sprint(l.Orders)})
}

// OrderBook is a depth snapshot. Asks ascend and bids descend by price.
// PrevSeqID and Checksum are set only on WebSocket pushes.
type OrderBook struct {
	Asks      []BookLevel `json:"asks"`
	Bids      []BookLevel `json:"bids"`
	Ts        Time        `json:"ts"`
	SeqID     int64       `json:"seqId"`
	PrevSeqID int64       `json:"prevSeqId"`
	Checksum  int64       `json:"checksum"`
}

type bookQuery struct {
	InstID string `json:"instId"`
	Sz     int    `json:"sz"`
}

type limitQuery struct {
	InstID string `json:"instId"`
	Limit  int    `json:"limit"`
}

// OrderBook returns up to depth levels per side (max 400).
func (s *MarketService) OrderBook(ctx context.Context, instID string, depth int) (OrderBook, error) {
	return one[OrderBook](ctx, s.c, publicGet("/api/v5/market/books", bookQuery{instID, depth}))
}

// FullOrderBook returns up to depth levels per side (max 5000).
func (s *MarketService) FullOrderBook(ctx context.Context, instID string, depth int) (OrderBook, error) {
	return one[OrderBook](ctx, s.c, publicGet("/api/v5/market/books-full", bookQuery{instID, depth}))
}

// Candle is one OHLCV bar. Index and mark price candles carry no volume.
type Candle struct {
	Ts          Time
	Open        Number
	High        Number
	Low         Number
	Close       Number
	Vol         Number
	VolCcy      Number
	VolCcyQuote Number
	Confirmed   bool
}

func (c *Candle) UnmarshalJSON(b []byte) error {
	var f []string
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	if len(f) < 5 {
		return fmt.Errorf("okx: candle has %d fields", len(f))
	}
	if err := c.Ts.UnmarshalJSON([]byte(f[0])); err != nil {
		return err
	}
	c.Open, c.High, c.Low, c.Close = Number(f[1]), Number(f[2]), Number(f[3]), Number(f[4])
	c.Vol, c.VolCcy, c.VolCcyQuote = "", "", ""
	switch len(f) {
	case 9:
		c.Vol, c.VolCcy, c.VolCcyQuote = Number(f[5]), Number(f[6]), Number(f[7])
	case 6:
	default:
		return fmt.Errorf("okx: candle has %d fields", len(f))
	}
	c.Confirmed = f[len(f)-1] == "1"
	return nil
}

type CandlesRequest struct {
	InstID string `json:"instId"`
	Bar    Bar    `json:"bar,omitempty"`
	// After returns candles older than this time; Before returns newer ones.
	After  Time `json:"after,omitempty"`
	Before Time `json:"before,omitempty"`
	Limit  int  `json:"limit,omitempty"`
}

// Candles returns recent candles, newest first (max 300 per request, up to
// 1440 most recent bars).
func (s *MarketService) Candles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/candles", req))
}

// HistoryCandles returns candles from recent years, newest first (max 100).
func (s *MarketService) HistoryCandles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/history-candles", req))
}

// AllHistoryCandles walks history backwards from req.After (or now) until
// req.Before or the start of history, newest first.
func (s *MarketService) AllHistoryCandles(ctx context.Context, req CandlesRequest) iter.Seq2[Candle, error] {
	return paginate(ctx, func(cursor Candle, hasCursor bool) ([]Candle, error) {
		if hasCursor {
			req.After = cursor.Ts
		}
		return s.HistoryCandles(ctx, req)
	})
}

// IndexCandles returns index price candles. InstID is an index like "BTC-USD".
func (s *MarketService) IndexCandles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/index-candles", req))
}

func (s *MarketService) HistoryIndexCandles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/history-index-candles", req))
}

func (s *MarketService) MarkPriceCandles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/mark-price-candles", req))
}

func (s *MarketService) HistoryMarkPriceCandles(ctx context.Context, req CandlesRequest) ([]Candle, error) {
	return list[Candle](ctx, s.c, publicGet("/api/v5/market/history-mark-price-candles", req))
}

type Trade struct {
	InstID  string `json:"instId"`
	TradeID string `json:"tradeId"`
	Px      Number `json:"px"`
	Sz      Number `json:"sz"`
	Side    Side   `json:"side"`
	Count   Number `json:"count,omitempty"`
	Source  string `json:"source"`
	SeqID   int64  `json:"seqId,omitempty"`
	Ts      Time   `json:"ts"`
}

// Trades returns the most recent trades (limit max 500).
func (s *MarketService) Trades(ctx context.Context, instID string, limit int) ([]Trade, error) {
	return list[Trade](ctx, s.c, publicGet("/api/v5/market/trades", limitQuery{instID, limit}))
}

type HistoryTradesRequest struct {
	InstID string `json:"instId"`
	// Type selects the cursor: "1" trade ID (default), "2" timestamp.
	Type   string `json:"type,omitempty"`
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// HistoryTrades returns trades from the last 3 months (max 100).
func (s *MarketService) HistoryTrades(ctx context.Context, req HistoryTradesRequest) ([]Trade, error) {
	return list[Trade](ctx, s.c, publicGet("/api/v5/market/history-trades", req))
}

type PlatformVolume struct {
	VolUsd Number `json:"volUsd"`
	VolCny Number `json:"volCny"`
	Ts     Time   `json:"ts"`
}

// PlatformVolume24h returns the 24h trading volume of the whole platform.
func (s *MarketService) PlatformVolume24h(ctx context.Context) (PlatformVolume, error) {
	return one[PlatformVolume](ctx, s.c, publicGet("/api/v5/market/platform-24-volume", nil))
}

type CallAuction struct {
	InstID         string `json:"instId"`
	EqPx           Number `json:"eqPx"`
	MatchedSz      Number `json:"matchedSz"`
	UnmatchedSz    Number `json:"unmatchedSz"`
	AuctionEndTime Time   `json:"auctionEndTime"`
	State          string `json:"state"`
	Ts             Time   `json:"ts"`
}

// CallAuction returns call auction details of a newly listing instrument.
func (s *MarketService) CallAuction(ctx context.Context, instID string) (CallAuction, error) {
	return one[CallAuction](ctx, s.c, publicGet("/api/v5/market/call-auction-details", map[string]string{"instId": instID}))
}
