package okx

import (
	"context"
	"iter"
	"strings"
)

// AccountService covers /api/v5/account: trading account balance, positions,
// leverage, fees and bills.
type AccountService struct{ c *Client }

type Balance struct {
	TotalEq     Number          `json:"totalEq"`
	IsoEq       Number          `json:"isoEq"`
	AdjEq       Number          `json:"adjEq"`
	AvailEq     Number          `json:"availEq"`
	OrdFroz     Number          `json:"ordFroz"`
	Imr         Number          `json:"imr"`
	Mmr         Number          `json:"mmr"`
	BorrowFroz  Number          `json:"borrowFroz"`
	MgnRatio    Number          `json:"mgnRatio"`
	NotionalUsd Number          `json:"notionalUsd"`
	Upl         Number          `json:"upl"`
	UTime       Time            `json:"uTime"`
	Details     []BalanceDetail `json:"details"`
}

type BalanceDetail struct {
	Ccy           string `json:"ccy"`
	Eq            Number `json:"eq"`
	CashBal       Number `json:"cashBal"`
	IsoEq         Number `json:"isoEq"`
	AvailEq       Number `json:"availEq"`
	DisEq         Number `json:"disEq"`
	FixedBal      Number `json:"fixedBal"`
	AvailBal      Number `json:"availBal"`
	FrozenBal     Number `json:"frozenBal"`
	OrdFrozen     Number `json:"ordFrozen"`
	Liab          Number `json:"liab"`
	Upl           Number `json:"upl"`
	UplLiab       Number `json:"uplLiab"`
	CrossLiab     Number `json:"crossLiab"`
	IsoLiab       Number `json:"isoLiab"`
	MgnRatio      Number `json:"mgnRatio"`
	Interest      Number `json:"interest"`
	Twap          Number `json:"twap"`
	MaxLoan       Number `json:"maxLoan"`
	EqUsd         Number `json:"eqUsd"`
	BorrowFroz    Number `json:"borrowFroz"`
	NotionalLever Number `json:"notionalLever"`
	StgyEq        Number `json:"stgyEq"`
	IsoUpl        Number `json:"isoUpl"`
	SpotInUseAmt  Number `json:"spotInUseAmt"`
	SpotBal       Number `json:"spotBal"`
	OpenAvgPx     Number `json:"openAvgPx"`
	AccAvgPx      Number `json:"accAvgPx"`
	SpotUpl       Number `json:"spotUpl"`
	SpotUplRatio  Number `json:"spotUplRatio"`
	TotalPnl      Number `json:"totalPnl"`
	TotalPnlRatio Number `json:"totalPnlRatio"`
	Imr           Number `json:"imr"`
	Mmr           Number `json:"mmr"`
	UTime         Time   `json:"uTime"`
}

// Balance returns the trading account balance, optionally only for the
// given currencies (up to 20).
func (s *AccountService) Balance(ctx context.Context, ccys ...string) (Balance, error) {
	return one[Balance](ctx, s.c, privateGet("/api/v5/account/balance", ccyQuery(ccys)))
}

type Position struct {
	InstType       InstType         `json:"instType"`
	InstID         string           `json:"instId"`
	MgnMode        MgnMode          `json:"mgnMode"`
	PosID          string           `json:"posId"`
	PosSide        PosSide          `json:"posSide"`
	Pos            Number           `json:"pos"`
	PosCcy         string           `json:"posCcy"`
	AvailPos       Number           `json:"availPos"`
	AvgPx          Number           `json:"avgPx"`
	NonSettleAvgPx Number           `json:"nonSettleAvgPx"`
	MarkPx         Number           `json:"markPx"`
	IdxPx          Number           `json:"idxPx"`
	Last           Number           `json:"last"`
	BePx           Number           `json:"bePx"`
	LiqPx          Number           `json:"liqPx"`
	Lever          Number           `json:"lever"`
	Upl            Number           `json:"upl"`
	UplRatio       Number           `json:"uplRatio"`
	UplLastPx      Number           `json:"uplLastPx"`
	UplRatioLastPx Number           `json:"uplRatioLastPx"`
	RealizedPnl    Number           `json:"realizedPnl"`
	SettledPnl     Number           `json:"settledPnl"`
	Pnl            Number           `json:"pnl"`
	Fee            Number           `json:"fee"`
	FundingFee     Number           `json:"fundingFee"`
	LiqPenalty     Number           `json:"liqPenalty"`
	Margin         Number           `json:"margin"`
	MgnRatio       Number           `json:"mgnRatio"`
	Imr            Number           `json:"imr"`
	Mmr            Number           `json:"mmr"`
	Liab           Number           `json:"liab"`
	LiabCcy        string           `json:"liabCcy"`
	Interest       Number           `json:"interest"`
	NotionalUsd    Number           `json:"notionalUsd"`
	Adl            string           `json:"adl"`
	Ccy            string           `json:"ccy"`
	TradeID        string           `json:"tradeId"`
	OptVal         Number           `json:"optVal"`
	DeltaBS        Number           `json:"deltaBS"`
	DeltaPA        Number           `json:"deltaPA"`
	GammaBS        Number           `json:"gammaBS"`
	GammaPA        Number           `json:"gammaPA"`
	ThetaBS        Number           `json:"thetaBS"`
	ThetaPA        Number           `json:"thetaPA"`
	VegaBS         Number           `json:"vegaBS"`
	VegaPA         Number           `json:"vegaPA"`
	CloseOrderAlgo []CloseOrderAlgo `json:"closeOrderAlgo"`
	CTime          Time             `json:"cTime"`
	UTime          Time             `json:"uTime"`
	PTime          Time             `json:"pTime"`
}

// CloseOrderAlgo is a TP/SL attached to a whole position.
type CloseOrderAlgo struct {
	AlgoID          string        `json:"algoId"`
	SlTriggerPx     Number        `json:"slTriggerPx"`
	SlTriggerPxType TriggerPxType `json:"slTriggerPxType"`
	TpTriggerPx     Number        `json:"tpTriggerPx"`
	TpTriggerPxType TriggerPxType `json:"tpTriggerPxType"`
	CloseFraction   Number        `json:"closeFraction"`
}

type PositionsRequest struct {
	InstType InstType `json:"instType,omitempty"`
	// InstID accepts up to 10 comma-separated instruments.
	InstID string `json:"instId,omitempty"`
	PosID  string `json:"posId,omitempty"`
}

// Positions returns open positions.
func (s *AccountService) Positions(ctx context.Context, req PositionsRequest) ([]Position, error) {
	return list[Position](ctx, s.c, privateGet("/api/v5/account/positions", req))
}

type PositionHistory struct {
	InstType InstType `json:"instType"`
	InstID   string   `json:"instId"`
	MgnMode  MgnMode  `json:"mgnMode"`
	// Type is how the position closed: 1 partially, 2 fully, 3 liquidation,
	// 4 partial liquidation, 5 ADL, 6 ADL partial.
	Type           string  `json:"type"`
	PosID          string  `json:"posId"`
	PosSide        PosSide `json:"posSide"`
	Direction      string  `json:"direction"`
	Lever          Number  `json:"lever"`
	OpenAvgPx      Number  `json:"openAvgPx"`
	NonSettleAvgPx Number  `json:"nonSettleAvgPx"`
	CloseAvgPx     Number  `json:"closeAvgPx"`
	OpenMaxPos     Number  `json:"openMaxPos"`
	CloseTotalPos  Number  `json:"closeTotalPos"`
	RealizedPnl    Number  `json:"realizedPnl"`
	SettledPnl     Number  `json:"settledPnl"`
	PnlRatio       Number  `json:"pnlRatio"`
	Pnl            Number  `json:"pnl"`
	Fee            Number  `json:"fee"`
	FundingFee     Number  `json:"fundingFee"`
	LiqPenalty     Number  `json:"liqPenalty"`
	TriggerPx      Number  `json:"triggerPx"`
	Uly            string  `json:"uly"`
	Ccy            string  `json:"ccy"`
	CTime          Time    `json:"cTime"`
	UTime          Time    `json:"uTime"`
}

type PositionsHistoryRequest struct {
	InstType InstType `json:"instType,omitempty"`
	InstID   string   `json:"instId,omitempty"`
	MgnMode  MgnMode  `json:"mgnMode,omitempty"`
	Type     string   `json:"type,omitempty"`
	PosID    string   `json:"posId,omitempty"`
	// After returns positions updated before this time; Before after it.
	After  Time `json:"after,omitempty"`
	Before Time `json:"before,omitempty"`
	Limit  int  `json:"limit,omitempty"`
}

// PositionsHistory returns positions closed in the last 3 months, newest
// first (max 100).
func (s *AccountService) PositionsHistory(ctx context.Context, req PositionsHistoryRequest) ([]PositionHistory, error) {
	return list[PositionHistory](ctx, s.c, privateGet("/api/v5/account/positions-history", req))
}

// AllPositionsHistory iterates over every closed position of the last 3
// months matching req, newest first.
func (s *AccountService) AllPositionsHistory(ctx context.Context, req PositionsHistoryRequest) iter.Seq2[PositionHistory, error] {
	return paginate(ctx, func(cursor PositionHistory, hasCursor bool) ([]PositionHistory, error) {
		if hasCursor {
			req.After = cursor.UTime
		}
		return s.PositionsHistory(ctx, req)
	})
}

type AccountConfig struct {
	UID     string `json:"uid"`
	MainUID string `json:"mainUid"`
	Label   string `json:"label"`
	// AcctLv: 1 spot, 2 futures, 3 multi-currency margin, 4 portfolio margin.
	AcctLv           string   `json:"acctLv"`
	PosMode          PosMode  `json:"posMode"`
	AutoLoan         bool     `json:"autoLoan"`
	GreeksType       string   `json:"greeksType"`
	Level            string   `json:"level"`
	LevelTmp         string   `json:"levelTmp"`
	CtIsoMode        string   `json:"ctIsoMode"`
	MgnIsoMode       string   `json:"mgnIsoMode"`
	RoleType         string   `json:"roleType"`
	SpotRoleType     string   `json:"spotRoleType"`
	OpAuth           string   `json:"opAuth"`
	KycLv            string   `json:"kycLv"`
	IP               string   `json:"ip"`
	Perm             string   `json:"perm"`
	AcctStpMode      string   `json:"acctStpMode"`
	EnableSpotBorrow bool     `json:"enableSpotBorrow"`
	SettleCcy        string   `json:"settleCcy"`
	SettleCcyList    []string `json:"settleCcyList"`
}

// Permissions returns the API key permissions: "read_only", "trade",
// "withdraw".
func (a AccountConfig) Permissions() []string {
	if a.Perm == "" {
		return nil
	}
	return strings.Split(a.Perm, ",")
}

// Config returns account settings and the permissions of the API key.
func (s *AccountService) Config(ctx context.Context) (AccountConfig, error) {
	return one[AccountConfig](ctx, s.c, privateGet("/api/v5/account/config", nil))
}

type Leverage struct {
	InstID  string  `json:"instId"`
	Ccy     string  `json:"ccy"`
	MgnMode MgnMode `json:"mgnMode"`
	PosSide PosSide `json:"posSide"`
	Lever   Number  `json:"lever"`
}

type SetLeverageRequest struct {
	InstID  string  `json:"instId,omitempty"`
	Ccy     string  `json:"ccy,omitempty"`
	Lever   Number  `json:"lever"`
	MgnMode MgnMode `json:"mgnMode"`
	PosSide PosSide `json:"posSide,omitempty"`
}

func (s *AccountService) SetLeverage(ctx context.Context, req SetLeverageRequest) (Leverage, error) {
	return one[Leverage](ctx, s.c, privatePost("/api/v5/account/set-leverage", req))
}

type LeverageInfoRequest struct {
	// InstID accepts up to 20 comma-separated instruments.
	InstID  string  `json:"instId,omitempty"`
	Ccy     string  `json:"ccy,omitempty"`
	MgnMode MgnMode `json:"mgnMode"`
}

func (s *AccountService) LeverageInfo(ctx context.Context, req LeverageInfoRequest) ([]Leverage, error) {
	return list[Leverage](ctx, s.c, privateGet("/api/v5/account/leverage-info", req))
}

// SetPositionMode switches between long/short and net mode. It fails while
// positions or orders are open.
func (s *AccountService) SetPositionMode(ctx context.Context, mode PosMode) error {
	_, err := list[struct{}](ctx, s.c, privatePost("/api/v5/account/set-position-mode", map[string]PosMode{"posMode": mode}))
	return err
}

type MaxSize struct {
	InstID  string `json:"instId"`
	Ccy     string `json:"ccy"`
	MaxBuy  Number `json:"maxBuy"`
	MaxSell Number `json:"maxSell"`
}

type MaxSizeRequest struct {
	// InstID accepts up to 5 comma-separated instruments.
	InstID        string `json:"instId"`
	TdMode        TdMode `json:"tdMode"`
	Ccy           string `json:"ccy,omitempty"`
	Px            Number `json:"px,omitempty"`
	Leverage      Number `json:"leverage,omitempty"`
	TradeQuoteCcy string `json:"tradeQuoteCcy,omitempty"`
}

// MaxOrderSize returns the maximum quantity that can be bought or sold.
func (s *AccountService) MaxOrderSize(ctx context.Context, req MaxSizeRequest) ([]MaxSize, error) {
	return list[MaxSize](ctx, s.c, privateGet("/api/v5/account/max-size", req))
}

type MaxAvailSize struct {
	InstID    string `json:"instId"`
	AvailBuy  Number `json:"availBuy"`
	AvailSell Number `json:"availSell"`
}

type MaxAvailSizeRequest struct {
	InstID        string `json:"instId"`
	TdMode        TdMode `json:"tdMode"`
	Ccy           string `json:"ccy,omitempty"`
	ReduceOnly    bool   `json:"reduceOnly,omitempty"`
	Px            Number `json:"px,omitempty"`
	TradeQuoteCcy string `json:"tradeQuoteCcy,omitempty"`
}

// MaxAvailSize returns the available balance (spot) or equity (margin) for
// opening positions.
func (s *AccountService) MaxAvailSize(ctx context.Context, req MaxAvailSizeRequest) ([]MaxAvailSize, error) {
	return list[MaxAvailSize](ctx, s.c, privateGet("/api/v5/account/max-avail-size", req))
}

// TradeFee rates are negative for commission paid and positive for rebates.
type TradeFee struct {
	InstType  InstType   `json:"instType"`
	Level     string     `json:"level"`
	Maker     Number     `json:"maker"`
	Taker     Number     `json:"taker"`
	MakerU    Number     `json:"makerU"`
	TakerU    Number     `json:"takerU"`
	MakerUSDC Number     `json:"makerUSDC"`
	TakerUSDC Number     `json:"takerUSDC"`
	RpiMaker  Number     `json:"rpiMaker"`
	Delivery  Number     `json:"delivery"`
	Exercise  Number     `json:"exercise"`
	RuleType  string     `json:"ruleType"`
	FeeGroup  []FeeGroup `json:"feeGroup"`
	Ts        Time       `json:"ts"`
}

type FeeGroup struct {
	GroupID string `json:"groupId"`
	Maker   Number `json:"maker"`
	Taker   Number `json:"taker"`
}

type TradeFeeRequest struct {
	InstType   InstType `json:"instType"`
	InstID     string   `json:"instId,omitempty"`
	InstFamily string   `json:"instFamily,omitempty"`
	RuleType   string   `json:"ruleType,omitempty"`
}

func (s *AccountService) TradeFee(ctx context.Context, req TradeFeeRequest) (TradeFee, error) {
	return one[TradeFee](ctx, s.c, privateGet("/api/v5/account/trade-fee", req))
}

type Bill struct {
	BillID    string   `json:"billId"`
	InstType  InstType `json:"instType"`
	InstID    string   `json:"instId"`
	Ccy       string   `json:"ccy"`
	MgnMode   string   `json:"mgnMode"`
	Type      string   `json:"type"`
	SubType   string   `json:"subType"`
	Bal       Number   `json:"bal"`
	BalChg    Number   `json:"balChg"`
	PosBal    Number   `json:"posBal"`
	PosBalChg Number   `json:"posBalChg"`
	Sz        Number   `json:"sz"`
	Px        Number   `json:"px"`
	Pnl       Number   `json:"pnl"`
	Fee       Number   `json:"fee"`
	Interest  Number   `json:"interest"`
	ExecType  string   `json:"execType"`
	OrdID     string   `json:"ordId"`
	ClOrdID   string   `json:"clOrdId"`
	TradeID   string   `json:"tradeId"`
	Tag       string   `json:"tag"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Notes     string   `json:"notes"`
	FillTime  Time     `json:"fillTime"`
	Ts        Time     `json:"ts"`
}

type BillsRequest struct {
	InstType InstType `json:"instType,omitempty"`
	InstID   string   `json:"instId,omitempty"`
	Ccy      string   `json:"ccy,omitempty"`
	MgnMode  MgnMode  `json:"mgnMode,omitempty"`
	CtType   string   `json:"ctType,omitempty"`
	Type     string   `json:"type,omitempty"`
	SubType  string   `json:"subType,omitempty"`
	// After and Before are bill IDs; Begin and End bound the time range.
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
	Begin  Time   `json:"begin,omitempty"`
	End    Time   `json:"end,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// Bills returns balance changes of the last 7 days, newest first.
func (s *AccountService) Bills(ctx context.Context, req BillsRequest) ([]Bill, error) {
	return list[Bill](ctx, s.c, privateGet("/api/v5/account/bills", req))
}

// BillsArchive returns balance changes of the last 3 months, newest first.
func (s *AccountService) BillsArchive(ctx context.Context, req BillsRequest) ([]Bill, error) {
	return list[Bill](ctx, s.c, privateGet("/api/v5/account/bills-archive", req))
}

// AllBills iterates over every bill of the last 3 months matching req.
func (s *AccountService) AllBills(ctx context.Context, req BillsRequest) iter.Seq2[Bill, error] {
	return paginate(ctx, func(cursor Bill, hasCursor bool) ([]Bill, error) {
		if hasCursor {
			req.After = cursor.BillID
		}
		return s.BillsArchive(ctx, req)
	})
}

type MarginAdjustment struct {
	InstID   string  `json:"instId"`
	PosSide  PosSide `json:"posSide"`
	Amt      Number  `json:"amt"`
	Type     string  `json:"type"`
	Leverage Number  `json:"leverage"`
	Ccy      string  `json:"ccy"`
}

type AdjustMarginRequest struct {
	InstID  string  `json:"instId"`
	PosSide PosSide `json:"posSide"`
	// Type is "add" or "reduce".
	Type string `json:"type"`
	Amt  Number `json:"amt"`
	Ccy  string `json:"ccy,omitempty"`
}

// AdjustMargin adds or removes margin of an isolated position.
func (s *AccountService) AdjustMargin(ctx context.Context, req AdjustMarginRequest) (MarginAdjustment, error) {
	return one[MarginAdjustment](ctx, s.c, privatePost("/api/v5/account/position/margin-balance", req))
}

type MaxWithdrawal struct {
	Ccy   string `json:"ccy"`
	MaxWd Number `json:"maxWd"`
	// MaxWdEx includes borrowing in auto-borrow mode.
	MaxWdEx Number `json:"maxWdEx"`
}

// MaxWithdrawal returns how much can be moved out of the trading account.
func (s *AccountService) MaxWithdrawal(ctx context.Context, ccys ...string) ([]MaxWithdrawal, error) {
	return list[MaxWithdrawal](ctx, s.c, privateGet("/api/v5/account/max-withdrawal", ccyQuery(ccys)))
}

func ccyQuery(ccys []string) map[string]string {
	return map[string]string{"ccy": strings.Join(ccys, ",")}
}
