package okx

import (
	"context"
	"time"
)

// PublicService covers /api/v5/public, /api/v5/system and trading statistics:
// instruments, funding, open interest, liquidations and server time.
type PublicService struct{ c *Client }

type Instrument struct {
	InstType          InstType `json:"instType"`
	InstID            string   `json:"instId"`
	InstIDCode        int64    `json:"instIdCode"`
	InstFamily        string   `json:"instFamily"`
	Uly               string   `json:"uly"`
	BaseCcy           string   `json:"baseCcy"`
	QuoteCcy          string   `json:"quoteCcy"`
	SettleCcy         string   `json:"settleCcy"`
	CtVal             Number   `json:"ctVal"`
	CtMult            Number   `json:"ctMult"`
	CtValCcy          string   `json:"ctValCcy"`
	CtType            string   `json:"ctType"`
	OptType           string   `json:"optType"`
	Stk               Number   `json:"stk"`
	ListTime          Time     `json:"listTime"`
	ExpTime           Time     `json:"expTime"`
	ContTdSwTime      Time     `json:"contTdSwTime"`
	Lever             Number   `json:"lever"`
	TickSz            Number   `json:"tickSz"`
	LotSz             Number   `json:"lotSz"`
	MinSz             Number   `json:"minSz"`
	MaxLmtSz          Number   `json:"maxLmtSz"`
	MaxMktSz          Number   `json:"maxMktSz"`
	MaxLmtAmt         Number   `json:"maxLmtAmt"`
	MaxMktAmt         Number   `json:"maxMktAmt"`
	MaxTwapSz         Number   `json:"maxTwapSz"`
	MaxIcebergSz      Number   `json:"maxIcebergSz"`
	MaxTriggerSz      Number   `json:"maxTriggerSz"`
	MaxStopSz         Number   `json:"maxStopSz"`
	PosLmtAmt         Number   `json:"posLmtAmt"`
	PosLmtPct         Number   `json:"posLmtPct"`
	InitPxLmtPct      Number   `json:"initPxLmtPct"`
	FloatPxLmtPct     Number   `json:"floatPxLmtPct"`
	MaxPxLmtPct       Number   `json:"maxPxLmtPct"`
	State             string   `json:"state"`
	RuleType          string   `json:"ruleType"`
	OpenType          string   `json:"openType"`
	GroupID           string   `json:"groupId"`
	TradeQuoteCcyList []string `json:"tradeQuoteCcyList"`
}

type InstrumentsRequest struct {
	InstType   InstType `json:"instType"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
}

// Instruments returns tradable instruments with their tick, lot and
// contract sizes.
func (s *PublicService) Instruments(ctx context.Context, req InstrumentsRequest) ([]Instrument, error) {
	return list[Instrument](ctx, s.c, publicGet("/api/v5/public/instruments", req))
}

// Time returns the OKX server time.
func (s *PublicService) Time(ctx context.Context) (time.Time, error) {
	t, err := one[struct {
		Ts Time `json:"ts"`
	}](ctx, s.c, publicGet("/api/v5/public/time", nil))
	return t.Ts.Time, err
}

type FundingRate struct {
	InstType        InstType `json:"instType"`
	InstID          string   `json:"instId"`
	Method          string   `json:"method"`
	FormulaType     string   `json:"formulaType"`
	FundingRate     Number   `json:"fundingRate"`
	NextFundingRate Number   `json:"nextFundingRate"`
	FundingTime     Time     `json:"fundingTime"`
	NextFundingTime Time     `json:"nextFundingTime"`
	PrevFundingTime Time     `json:"prevFundingTime"`
	MinFundingRate  Number   `json:"minFundingRate"`
	MaxFundingRate  Number   `json:"maxFundingRate"`
	SettFundingRate Number   `json:"settFundingRate"`
	SettState       string   `json:"settState"`
	Premium         Number   `json:"premium"`
	InterestRate    Number   `json:"interestRate"`
	ImpactValue     Number   `json:"impactValue"`
	Ts              Time     `json:"ts"`
}

// FundingRate returns the current funding rate of a perpetual swap, e.g.
// "BTC-USDT-SWAP".
func (s *PublicService) FundingRate(ctx context.Context, instID string) (FundingRate, error) {
	return one[FundingRate](ctx, s.c, publicGet("/api/v5/public/funding-rate", map[string]string{"instId": instID}))
}

type FundingRateHistory struct {
	InstType     InstType `json:"instType"`
	InstID       string   `json:"instId"`
	Method       string   `json:"method"`
	FormulaType  string   `json:"formulaType"`
	FundingRate  Number   `json:"fundingRate"`
	RealizedRate Number   `json:"realizedRate"`
	FundingTime  Time     `json:"fundingTime"`
}

type FundingRateHistoryRequest struct {
	InstID string `json:"instId"`
	// After returns records older than this funding time; Before newer ones.
	After  Time `json:"after,omitempty"`
	Before Time `json:"before,omitempty"`
	Limit  int  `json:"limit,omitempty"`
}

// FundingRateHistory returns settled funding rates of the last 3 months,
// newest first (max 400).
func (s *PublicService) FundingRateHistory(ctx context.Context, req FundingRateHistoryRequest) ([]FundingRateHistory, error) {
	return list[FundingRateHistory](ctx, s.c, publicGet("/api/v5/public/funding-rate-history", req))
}

type OpenInterest struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	Oi       Number   `json:"oi"`
	OiCcy    Number   `json:"oiCcy"`
	OiUsd    Number   `json:"oiUsd"`
	Ts       Time     `json:"ts"`
}

type OpenInterestRequest struct {
	InstType   InstType `json:"instType"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
}

func (s *PublicService) OpenInterest(ctx context.Context, req OpenInterestRequest) ([]OpenInterest, error) {
	return list[OpenInterest](ctx, s.c, publicGet("/api/v5/public/open-interest", req))
}

type MarkPrice struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	MarkPx   Number   `json:"markPx"`
	Ts       Time     `json:"ts"`
}

type MarkPriceRequest struct {
	InstType   InstType `json:"instType"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
}

func (s *PublicService) MarkPrice(ctx context.Context, req MarkPriceRequest) ([]MarkPrice, error) {
	return list[MarkPrice](ctx, s.c, publicGet("/api/v5/public/mark-price", req))
}

type PriceLimit struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	BuyLmt   Number   `json:"buyLmt"`
	SellLmt  Number   `json:"sellLmt"`
	Enabled  bool     `json:"enabled"`
	Ts       Time     `json:"ts"`
}

// PriceLimit returns the highest buy and lowest sell price currently allowed.
func (s *PublicService) PriceLimit(ctx context.Context, instID string) (PriceLimit, error) {
	return one[PriceLimit](ctx, s.c, publicGet("/api/v5/public/price-limit", map[string]string{"instId": instID}))
}

type EstimatedPrice struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	SettlePx Number   `json:"settlePx"`
	Ts       Time     `json:"ts"`
}

// EstimatedPrice returns the estimated delivery or exercise price within one
// hour before settlement of futures and options.
func (s *PublicService) EstimatedPrice(ctx context.Context, instID string) (EstimatedPrice, error) {
	return one[EstimatedPrice](ctx, s.c, publicGet("/api/v5/public/estimated-price", map[string]string{"instId": instID}))
}

type Liquidation struct {
	InstType   InstType            `json:"instType"`
	InstID     string              `json:"instId"`
	InstFamily string              `json:"instFamily"`
	Uly        string              `json:"uly"`
	TotalLoss  Number              `json:"totalLoss"`
	Details    []LiquidationDetail `json:"details"`
}

type LiquidationDetail struct {
	Side    Side    `json:"side"`
	PosSide PosSide `json:"posSide"`
	BkPx    Number  `json:"bkPx"`
	Sz      Number  `json:"sz"`
	BkLoss  Number  `json:"bkLoss"`
	Ccy     string  `json:"ccy"`
	Ts      Time    `json:"ts"`
}

type LiquidationOrdersRequest struct {
	InstType   InstType `json:"instType"`
	MgnMode    MgnMode  `json:"mgnMode,omitempty"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
	Ccy        string   `json:"ccy,omitempty"`
	// State is "unfilled" or "filled"; required for SWAP and FUTURES.
	State  string `json:"state,omitempty"`
	Alias  string `json:"alias,omitempty"`
	After  Time   `json:"after,omitempty"`
	Before Time   `json:"before,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// LiquidationOrders returns liquidations of the last 7 days.
func (s *PublicService) LiquidationOrders(ctx context.Context, req LiquidationOrdersRequest) ([]Liquidation, error) {
	return list[Liquidation](ctx, s.c, publicGet("/api/v5/public/liquidation-orders", req))
}

type PositionTier struct {
	InstFamily   string `json:"instFamily"`
	InstID       string `json:"instId"`
	Uly          string `json:"uly"`
	Tier         string `json:"tier"`
	MinSz        Number `json:"minSz"`
	MaxSz        Number `json:"maxSz"`
	Mmr          Number `json:"mmr"`
	Imr          Number `json:"imr"`
	MaxLever     Number `json:"maxLever"`
	OptMgnFactor Number `json:"optMgnFactor"`
	QuoteMaxLoan Number `json:"quoteMaxLoan"`
	BaseMaxLoan  Number `json:"baseMaxLoan"`
}

type PositionTiersRequest struct {
	InstType   InstType `json:"instType"`
	TdMode     TdMode   `json:"tdMode"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
	Ccy        string   `json:"ccy,omitempty"`
	Tier       string   `json:"tier,omitempty"`
}

func (s *PublicService) PositionTiers(ctx context.Context, req PositionTiersRequest) ([]PositionTier, error) {
	return list[PositionTier](ctx, s.c, publicGet("/api/v5/public/position-tiers", req))
}

type ContractConversion struct {
	InstID string `json:"instId"`
	Type   string `json:"type"`
	Px     Number `json:"px"`
	Sz     Number `json:"sz"`
	Unit   string `json:"unit"`
}

type ConvertContractCoinRequest struct {
	// Type "1" converts currency to contracts, "2" contracts to currency.
	Type   string `json:"type,omitempty"`
	InstID string `json:"instId"`
	Sz     Number `json:"sz"`
	Px     Number `json:"px,omitempty"`
	// Unit "coin" (default) or "usds" for USDT/USDC-margined contracts.
	Unit   string `json:"unit,omitempty"`
	OpType string `json:"opType,omitempty"`
}

// ConvertContractCoin converts between contracts and coin amounts.
func (s *PublicService) ConvertContractCoin(ctx context.Context, req ConvertContractCoinRequest) (ContractConversion, error) {
	return one[ContractConversion](ctx, s.c, publicGet("/api/v5/public/convert-contract-coin", req))
}

type OptionSummary struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	Uly      string   `json:"uly"`
	Delta    Number   `json:"delta"`
	Gamma    Number   `json:"gamma"`
	Vega     Number   `json:"vega"`
	Theta    Number   `json:"theta"`
	DeltaBS  Number   `json:"deltaBS"`
	GammaBS  Number   `json:"gammaBS"`
	VegaBS   Number   `json:"vegaBS"`
	ThetaBS  Number   `json:"thetaBS"`
	Lever    Number   `json:"lever"`
	MarkVol  Number   `json:"markVol"`
	BidVol   Number   `json:"bidVol"`
	AskVol   Number   `json:"askVol"`
	RealVol  Number   `json:"realVol"`
	VolLv    Number   `json:"volLv"`
	FwdPx    Number   `json:"fwdPx"`
	Ts       Time     `json:"ts"`
}

type OptionSummaryRequest struct {
	InstFamily string `json:"instFamily"`
	ExpTime    string `json:"expTime,omitempty"`
}

// OptionSummary returns greeks and volatility of all options of a family,
// e.g. "BTC-USD".
func (s *PublicService) OptionSummary(ctx context.Context, req OptionSummaryRequest) ([]OptionSummary, error) {
	return list[OptionSummary](ctx, s.c, publicGet("/api/v5/public/opt-summary", req))
}

type SystemStatus struct {
	Title        string `json:"title"`
	State        string `json:"state"`
	Begin        Time   `json:"begin"`
	End          Time   `json:"end"`
	PreOpenBegin Time   `json:"preOpenBegin"`
	Href         string `json:"href"`
	ServiceType  string `json:"serviceType"`
	System       string `json:"system"`
	ScheDesc     string `json:"scheDesc"`
	MaintType    string `json:"maintType"`
	Env          string `json:"env"`
}

// SystemStatus returns scheduled and ongoing maintenance.
func (s *PublicService) SystemStatus(ctx context.Context) ([]SystemStatus, error) {
	return list[SystemStatus](ctx, s.c, publicGet("/api/v5/system/status", nil))
}

// TakerFlow is taker buy and sell volume over one period.
type TakerFlow struct {
	Ts      Time
	SellVol Number
	BuyVol  Number
}

func (t *TakerFlow) UnmarshalJSON(b []byte) error {
	var f [3]string
	if err := unmarshalArray(b, f[:]); err != nil {
		return err
	}
	t.SellVol, t.BuyVol = Number(f[1]), Number(f[2])
	return t.Ts.UnmarshalJSON([]byte(f[0]))
}

type TakerVolumeRequest struct {
	Ccy      string   `json:"ccy"`
	InstType InstType `json:"instType"`
	Begin    Time     `json:"begin,omitempty"`
	End      Time     `json:"end,omitempty"`
	// Period is "5m" (default), "1H" or "1D".
	Period string `json:"period,omitempty"`
}

// TakerVolume returns taker buy/sell volume of a currency, newest first.
func (s *PublicService) TakerVolume(ctx context.Context, req TakerVolumeRequest) ([]TakerFlow, error) {
	return list[TakerFlow](ctx, s.c, publicGet("/api/v5/rubik/stat/taker-volume", req))
}

// Ratio is a timestamped ratio from trading statistics.
type Ratio struct {
	Ts    Time
	Ratio Number
}

func (r *Ratio) UnmarshalJSON(b []byte) error {
	var f [2]string
	if err := unmarshalArray(b, f[:]); err != nil {
		return err
	}
	r.Ratio = Number(f[1])
	return r.Ts.UnmarshalJSON([]byte(f[0]))
}

type LongShortRatioRequest struct {
	Ccy    string `json:"ccy"`
	Begin  Time   `json:"begin,omitempty"`
	End    Time   `json:"end,omitempty"`
	Period string `json:"period,omitempty"`
}

// LongShortAccountRatio returns the ratio of accounts net long to net short
// on futures and swaps of a currency, newest first.
func (s *PublicService) LongShortAccountRatio(ctx context.Context, req LongShortRatioRequest) ([]Ratio, error) {
	return list[Ratio](ctx, s.c, publicGet("/api/v5/rubik/stat/contracts/long-short-account-ratio", req))
}
