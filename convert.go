package okx

import "context"

// ConvertCurrencies returns the currencies that can be converted.
func (s *AssetService) ConvertCurrencies(ctx context.Context) ([]string, error) {
	rows, err := list[struct {
		Ccy string `json:"ccy"`
	}](ctx, s.c, privateGet("/api/v5/asset/convert/currencies", nil))
	if err != nil {
		return nil, err
	}
	ccys := make([]string, len(rows))
	for i, r := range rows {
		ccys[i] = r.Ccy
	}
	return ccys, nil
}

type ConvertPair struct {
	InstID      string `json:"instId"`
	BaseCcy     string `json:"baseCcy"`
	BaseCcyMin  Number `json:"baseCcyMin"`
	BaseCcyMax  Number `json:"baseCcyMax"`
	QuoteCcy    string `json:"quoteCcy"`
	QuoteCcyMin Number `json:"quoteCcyMin"`
	QuoteCcyMax Number `json:"quoteCcyMax"`
}

type ConvertPairRequest struct {
	FromCcy string `json:"fromCcy"`
	ToCcy   string `json:"toCcy"`
	// ConvertMode: "0" standard (default), "1" large order convert for VIP.
	ConvertMode string `json:"convertMode,omitempty"`
}

// ConvertPair returns the pair for converting FromCcy into ToCcy, with the
// minimum and maximum amounts of each side.
func (s *AssetService) ConvertPair(ctx context.Context, req ConvertPairRequest) (ConvertPair, error) {
	return one[ConvertPair](ctx, s.c, privateGet("/api/v5/asset/convert/currency-pair", req))
}

type ConvertQuoteRequest struct {
	BaseCcy  string `json:"baseCcy"`
	QuoteCcy string `json:"quoteCcy"`
	// Side is relative to BaseCcy: SideBuy spends QuoteCcy to get BaseCcy.
	Side Side `json:"side"`
	// RfqSz is the amount to convert, in RfqSzCcy (BaseCcy or QuoteCcy).
	RfqSz       Number `json:"rfqSz"`
	RfqSzCcy    string `json:"rfqSzCcy"`
	ClQReqID    string `json:"clQReqId,omitempty"`
	Tag         string `json:"tag,omitempty"`
	ConvertMode string `json:"convertMode,omitempty"`
}

type ConvertQuoteResult struct {
	QuoteID   string `json:"quoteId"`
	ClQReqID  string `json:"clQReqId"`
	BaseCcy   string `json:"baseCcy"`
	QuoteCcy  string `json:"quoteCcy"`
	Side      Side   `json:"side"`
	OrigRfqSz Number `json:"origRfqSz"`
	RfqSz     Number `json:"rfqSz"`
	RfqSzCcy  string `json:"rfqSzCcy"`
	// CnvtPx is the price in QuoteCcy per BaseCcy.
	CnvtPx    Number `json:"cnvtPx"`
	BaseSz    Number `json:"baseSz"`
	QuoteSz   Number `json:"quoteSz"`
	QuoteTime Time   `json:"quoteTime"`
	// TTLMs is how long the quote stays valid, in milliseconds.
	TTLMs Number `json:"ttlMs"`
}

// ConvertQuote requests a quote. OKX allows one quote per pair every 5
// seconds; the client-side limiter waits for that when rate limiting is on.
func (s *AssetService) ConvertQuote(ctx context.Context, req ConvertQuoteRequest) (ConvertQuoteResult, error) {
	r := privatePost("/api/v5/asset/convert/estimate-quote", req)
	r.cost = map[string]int{req.BaseCcy + "-" + req.QuoteCcy: 1}
	return one[ConvertQuoteResult](ctx, s.c, r)
}

type ConvertRequest struct {
	QuoteID  string `json:"quoteId"`
	BaseCcy  string `json:"baseCcy"`
	QuoteCcy string `json:"quoteCcy"`
	Side     Side   `json:"side"`
	// Sz must not exceed the quoted RfqSz; SzCcy is the quote's RfqSzCcy.
	Sz          Number `json:"sz"`
	SzCcy       string `json:"szCcy"`
	ClTReqID    string `json:"clTReqId,omitempty"`
	Tag         string `json:"tag,omitempty"`
	ConvertMode string `json:"convertMode,omitempty"`
}

type ConvertTrade struct {
	TradeID  string `json:"tradeId"`
	QuoteID  string `json:"quoteId"`
	ClTReqID string `json:"clTReqId"`
	// State is "fullyFilled" or "rejected".
	State       string `json:"state"`
	InstID      string `json:"instId"`
	BaseCcy     string `json:"baseCcy"`
	QuoteCcy    string `json:"quoteCcy"`
	Side        Side   `json:"side"`
	FillPx      Number `json:"fillPx"`
	FillBaseSz  Number `json:"fillBaseSz"`
	FillQuoteSz Number `json:"fillQuoteSz"`
	Ts          Time   `json:"ts"`
}

// Convert swaps one currency for another in the funding account at a fixed
// quoted price, without placing an order on the order book. The flow is:
// ConvertPair for the allowed amounts, ConvertQuote for a quote, then Convert
// with its QuoteID before the quote expires (ConvertQuoteResult.TTLMs).
//
// Check State: a rejected convert is returned without an error.
func (s *AssetService) Convert(ctx context.Context, req ConvertRequest) (ConvertTrade, error) {
	return one[ConvertTrade](ctx, s.c, privatePost("/api/v5/asset/convert/trade", req))
}

type ConvertHistoryRequest struct {
	ClTReqID string `json:"clTReqId,omitempty"`
	After    Time   `json:"after,omitempty"`
	Before   Time   `json:"before,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// ConvertHistory returns past converts, newest first, up to 100 per page.
func (s *AssetService) ConvertHistory(ctx context.Context, req ConvertHistoryRequest) ([]ConvertTrade, error) {
	return list[ConvertTrade](ctx, s.c, privateGet("/api/v5/asset/convert/history", req))
}
