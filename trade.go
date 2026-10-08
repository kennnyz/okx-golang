package okx

import (
	"context"
	"crypto/rand"
	"iter"
	"strconv"
	"time"
)

// TradeService covers /api/v5/trade: orders, fills and algo orders.
type TradeService struct{ c *Client }

type PlaceOrderRequest struct {
	InstID  string  `json:"instId"`
	TdMode  TdMode  `json:"tdMode"`
	Side    Side    `json:"side"`
	OrdType OrdType `json:"ordType"`
	Sz      Number  `json:"sz"`
	Px      Number  `json:"px,omitempty"`
	PxUsd   Number  `json:"pxUsd,omitempty"`
	PxVol   Number  `json:"pxVol,omitempty"`
	PosSide PosSide `json:"posSide,omitempty"`
	Ccy     string  `json:"ccy,omitempty"`
	// ClOrdID is your idempotency key: up to 32 alphanumerics.
	ClOrdID        string            `json:"clOrdId,omitempty"`
	Tag            string            `json:"tag,omitempty"`
	ReduceOnly     bool              `json:"reduceOnly,omitempty"`
	TgtCcy         TgtCcy            `json:"tgtCcy,omitempty"`
	BanAmend       bool              `json:"banAmend,omitempty"`
	PxAmendType    string            `json:"pxAmendType,omitempty"`
	TradeQuoteCcy  string            `json:"tradeQuoteCcy,omitempty"`
	StpMode        string            `json:"stpMode,omitempty"`
	AttachAlgoOrds []AttachAlgoOrder `json:"attachAlgoOrds,omitempty"`
}

// AttachAlgoOrder is a take-profit and/or stop-loss attached to an order.
// An order price of "-1" executes at market.
type AttachAlgoOrder struct {
	AttachAlgoID         string        `json:"attachAlgoId,omitempty"`
	AttachAlgoClOrdID    string        `json:"attachAlgoClOrdId,omitempty"`
	TpTriggerPx          Number        `json:"tpTriggerPx,omitempty"`
	TpTriggerPxType      TriggerPxType `json:"tpTriggerPxType,omitempty"`
	TpOrdPx              Number        `json:"tpOrdPx,omitempty"`
	TpOrdKind            string        `json:"tpOrdKind,omitempty"`
	SlTriggerPx          Number        `json:"slTriggerPx,omitempty"`
	SlTriggerPxType      TriggerPxType `json:"slTriggerPxType,omitempty"`
	SlOrdPx              Number        `json:"slOrdPx,omitempty"`
	Sz                   Number        `json:"sz,omitempty"`
	AmendPxOnTriggerType string        `json:"amendPxOnTriggerType,omitempty"`
	FailCode             string        `json:"failCode,omitempty"`
	FailReason           string        `json:"failReason,omitempty"`
}

// OrderResult is the outcome of placing, amending or cancelling one order.
// In batch calls check Err on every item.
type OrderResult struct {
	OrdID   string `json:"ordId"`
	ClOrdID string `json:"clOrdId"`
	Tag     string `json:"tag"`
	ReqID   string `json:"reqId"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
	Ts      Time   `json:"ts"`
}

// Err returns the item's error, or nil if it succeeded.
func (r OrderResult) Err() error {
	if r.SCode == "" || r.SCode == "0" {
		return nil
	}
	return &APIError{Code: r.SCode, Msg: r.SMsg}
}

// PlaceOrder places one order. A rejected order returns an *APIError whose
// Items hold the reason, e.g. 51008 insufficient balance.
func (s *TradeService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (OrderResult, error) {
	return one[OrderResult](ctx, s.c, tradePost("/api/v5/trade/order", req, req.InstID))
}

// PlaceOrders places up to 20 orders. Orders succeed or fail individually:
// err is set only when the whole request failed, otherwise check each Err.
func (s *TradeService) PlaceOrders(ctx context.Context, reqs []PlaceOrderRequest) ([]OrderResult, error) {
	return list[OrderResult](ctx, s.c, batchPost("/api/v5/trade/batch-orders", reqs))
}

type CancelOrderRequest struct {
	InstID  string `json:"instId"`
	OrdID   string `json:"ordId,omitempty"`
	ClOrdID string `json:"clOrdId,omitempty"`
}

func (s *TradeService) CancelOrder(ctx context.Context, req CancelOrderRequest) (OrderResult, error) {
	return one[OrderResult](ctx, s.c, tradePost("/api/v5/trade/cancel-order", req, req.InstID))
}

// CancelOrders cancels up to 20 orders; check Err on each result.
func (s *TradeService) CancelOrders(ctx context.Context, reqs []CancelOrderRequest) ([]OrderResult, error) {
	return list[OrderResult](ctx, s.c, batchPost("/api/v5/trade/cancel-batch-orders", reqs))
}

type AmendOrderRequest struct {
	InstID         string                 `json:"instId"`
	OrdID          string                 `json:"ordId,omitempty"`
	ClOrdID        string                 `json:"clOrdId,omitempty"`
	ReqID          string                 `json:"reqId,omitempty"`
	CxlOnFail      bool                   `json:"cxlOnFail,omitempty"`
	NewSz          Number                 `json:"newSz,omitempty"`
	NewPx          Number                 `json:"newPx,omitempty"`
	NewPxUsd       Number                 `json:"newPxUsd,omitempty"`
	NewPxVol       Number                 `json:"newPxVol,omitempty"`
	AttachAlgoOrds []AmendAttachAlgoOrder `json:"attachAlgoOrds,omitempty"`
}

type AmendAttachAlgoOrder struct {
	AttachAlgoID         string        `json:"attachAlgoId,omitempty"`
	AttachAlgoClOrdID    string        `json:"attachAlgoClOrdId,omitempty"`
	NewTpTriggerPx       Number        `json:"newTpTriggerPx,omitempty"`
	NewTpTriggerPxType   TriggerPxType `json:"newTpTriggerPxType,omitempty"`
	NewTpOrdPx           Number        `json:"newTpOrdPx,omitempty"`
	NewTpOrdKind         string        `json:"newTpOrdKind,omitempty"`
	NewSlTriggerPx       Number        `json:"newSlTriggerPx,omitempty"`
	NewSlTriggerPxType   TriggerPxType `json:"newSlTriggerPxType,omitempty"`
	NewSlOrdPx           Number        `json:"newSlOrdPx,omitempty"`
	Sz                   Number        `json:"sz,omitempty"`
	AmendPxOnTriggerType string        `json:"amendPxOnTriggerType,omitempty"`
}

// AmendOrder changes the price or size of a pending order.
func (s *TradeService) AmendOrder(ctx context.Context, req AmendOrderRequest) (OrderResult, error) {
	return one[OrderResult](ctx, s.c, tradePost("/api/v5/trade/amend-order", req, req.InstID))
}

// AmendOrders amends up to 20 orders; check Err on each result.
func (s *TradeService) AmendOrders(ctx context.Context, reqs []AmendOrderRequest) ([]OrderResult, error) {
	return list[OrderResult](ctx, s.c, batchPost("/api/v5/trade/amend-batch-orders", reqs))
}

type ClosePositionRequest struct {
	InstID  string  `json:"instId"`
	MgnMode MgnMode `json:"mgnMode"`
	PosSide PosSide `json:"posSide,omitempty"`
	Ccy     string  `json:"ccy,omitempty"`
	// AutoCxl cancels pending orders that would block the close.
	AutoCxl bool   `json:"autoCxl,omitempty"`
	ClOrdID string `json:"clOrdId,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type ClosePositionResult struct {
	InstID  string  `json:"instId"`
	PosSide PosSide `json:"posSide"`
	ClOrdID string  `json:"clOrdId"`
	Tag     string  `json:"tag"`
}

// ClosePosition closes a whole position with a market order.
func (s *TradeService) ClosePosition(ctx context.Context, req ClosePositionRequest) (ClosePositionResult, error) {
	return one[ClosePositionResult](ctx, s.c, tradePost("/api/v5/trade/close-position", req, req.InstID))
}

type Order struct {
	InstType           InstType          `json:"instType"`
	InstID             string            `json:"instId"`
	OrdID              string            `json:"ordId"`
	ClOrdID            string            `json:"clOrdId"`
	Tag                string            `json:"tag"`
	Px                 Number            `json:"px"`
	PxUsd              Number            `json:"pxUsd"`
	PxVol              Number            `json:"pxVol"`
	PxType             string            `json:"pxType"`
	Sz                 Number            `json:"sz"`
	OrdType            OrdType           `json:"ordType"`
	Side               Side              `json:"side"`
	PosSide            PosSide           `json:"posSide"`
	TdMode             TdMode            `json:"tdMode"`
	TgtCcy             TgtCcy            `json:"tgtCcy"`
	Ccy                string            `json:"ccy"`
	State              OrderState        `json:"state"`
	Lever              Number            `json:"lever"`
	AccFillSz          Number            `json:"accFillSz"`
	AvgPx              Number            `json:"avgPx"`
	FillPx             Number            `json:"fillPx"`
	FillSz             Number            `json:"fillSz"`
	FillTime           Time              `json:"fillTime"`
	FillFee            Number            `json:"fillFee"`
	FillFeeCcy         string            `json:"fillFeeCcy"`
	FillPnl            Number            `json:"fillPnl"`
	ExecType           string            `json:"execType"`
	TradeID            string            `json:"tradeId"`
	Fee                Number            `json:"fee"`
	FeeCcy             string            `json:"feeCcy"`
	Rebate             Number            `json:"rebate"`
	RebateCcy          string            `json:"rebateCcy"`
	Pnl                Number            `json:"pnl"`
	Source             string            `json:"source"`
	Category           string            `json:"category"`
	ReduceOnly         Bool              `json:"reduceOnly"`
	StpMode            string            `json:"stpMode"`
	CancelSource       string            `json:"cancelSource"`
	CancelSourceReason string            `json:"cancelSourceReason"`
	AttachAlgoClOrdID  string            `json:"attachAlgoClOrdId"`
	AttachAlgoOrds     []AttachAlgoOrder `json:"attachAlgoOrds"`
	AlgoID             string            `json:"algoId"`
	AlgoClOrdID        string            `json:"algoClOrdId"`
	TradeQuoteCcy      string            `json:"tradeQuoteCcy"`
	// AmendResult, Code, Msg and ReqID are set on WebSocket order pushes.
	AmendResult string `json:"amendResult,omitempty"`
	Code        string `json:"code,omitempty"`
	Msg         string `json:"msg,omitempty"`
	ReqID       string `json:"reqId,omitempty"`
	CTime       Time   `json:"cTime"`
	UTime       Time   `json:"uTime"`
}

type OrderRequest struct {
	InstID  string `json:"instId"`
	OrdID   string `json:"ordId,omitempty"`
	ClOrdID string `json:"clOrdId,omitempty"`
}

// Order returns one order by ordId or clOrdId.
func (s *TradeService) Order(ctx context.Context, req OrderRequest) (Order, error) {
	r := privateGet("/api/v5/trade/order", req)
	r.cost = map[string]int{req.InstID: 1}
	return one[Order](ctx, s.c, r)
}

type PendingOrdersRequest struct {
	InstType   InstType   `json:"instType,omitempty"`
	InstFamily string     `json:"instFamily,omitempty"`
	InstID     string     `json:"instId,omitempty"`
	OrdType    OrdType    `json:"ordType,omitempty"`
	State      OrderState `json:"state,omitempty"`
	After      string     `json:"after,omitempty"`
	Before     string     `json:"before,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

// PendingOrders returns live and partially filled orders.
func (s *TradeService) PendingOrders(ctx context.Context, req PendingOrdersRequest) ([]Order, error) {
	return list[Order](ctx, s.c, privateGet("/api/v5/trade/orders-pending", req))
}

type OrdersHistoryRequest struct {
	InstType   InstType   `json:"instType"`
	InstFamily string     `json:"instFamily,omitempty"`
	InstID     string     `json:"instId,omitempty"`
	OrdType    OrdType    `json:"ordType,omitempty"`
	State      OrderState `json:"state,omitempty"`
	Category   string     `json:"category,omitempty"`
	// After and Before are order IDs; Begin and End bound the time range.
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
	Begin  Time   `json:"begin,omitempty"`
	End    Time   `json:"end,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// OrdersHistory returns completed orders of the last 7 days, newest first.
func (s *TradeService) OrdersHistory(ctx context.Context, req OrdersHistoryRequest) ([]Order, error) {
	return list[Order](ctx, s.c, privateGet("/api/v5/trade/orders-history", req))
}

// OrdersHistoryArchive returns completed orders of the last 3 months.
func (s *TradeService) OrdersHistoryArchive(ctx context.Context, req OrdersHistoryRequest) ([]Order, error) {
	return list[Order](ctx, s.c, privateGet("/api/v5/trade/orders-history-archive", req))
}

// AllOrdersHistory iterates over every completed order of the last 3 months
// matching req, newest first.
func (s *TradeService) AllOrdersHistory(ctx context.Context, req OrdersHistoryRequest) iter.Seq2[Order, error] {
	return paginate(ctx, func(cursor Order, hasCursor bool) ([]Order, error) {
		if hasCursor {
			req.After = cursor.OrdID
		}
		return s.OrdersHistoryArchive(ctx, req)
	})
}

type Fill struct {
	InstType      InstType `json:"instType"`
	InstID        string   `json:"instId"`
	TradeID       string   `json:"tradeId"`
	OrdID         string   `json:"ordId"`
	ClOrdID       string   `json:"clOrdId"`
	BillID        string   `json:"billId"`
	SubType       string   `json:"subType"`
	Tag           string   `json:"tag"`
	FillPx        Number   `json:"fillPx"`
	FillSz        Number   `json:"fillSz"`
	FillIdxPx     Number   `json:"fillIdxPx"`
	FillMarkPx    Number   `json:"fillMarkPx"`
	FillPnl       Number   `json:"fillPnl"`
	Side          Side     `json:"side"`
	PosSide       PosSide  `json:"posSide"`
	ExecType      string   `json:"execType"`
	Fee           Number   `json:"fee"`
	FeeCcy        string   `json:"feeCcy"`
	FeeRate       Number   `json:"feeRate"`
	TradeQuoteCcy string   `json:"tradeQuoteCcy"`
	FillTime      Time     `json:"fillTime"`
	Ts            Time     `json:"ts"`
}

type FillsRequest struct {
	InstType   InstType `json:"instType,omitempty"`
	InstFamily string   `json:"instFamily,omitempty"`
	InstID     string   `json:"instId,omitempty"`
	OrdID      string   `json:"ordId,omitempty"`
	SubType    string   `json:"subType,omitempty"`
	// After and Before are bill IDs; Begin and End bound the time range.
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
	Begin  Time   `json:"begin,omitempty"`
	End    Time   `json:"end,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// Fills returns executions of the last 3 days, newest first.
func (s *TradeService) Fills(ctx context.Context, req FillsRequest) ([]Fill, error) {
	return list[Fill](ctx, s.c, privateGet("/api/v5/trade/fills", req))
}

// FillsHistory returns executions of the last 3 months. InstType is required.
func (s *TradeService) FillsHistory(ctx context.Context, req FillsRequest) ([]Fill, error) {
	return list[Fill](ctx, s.c, privateGet("/api/v5/trade/fills-history", req))
}

// AllFills iterates over every execution of the last 3 months matching req,
// newest first. InstType is required.
func (s *TradeService) AllFills(ctx context.Context, req FillsRequest) iter.Seq2[Fill, error] {
	return paginate(ctx, func(cursor Fill, hasCursor bool) ([]Fill, error) {
		if hasCursor {
			req.After = cursor.BillID
		}
		return s.FillsHistory(ctx, req)
	})
}

type CancelAllAfterResult struct {
	TriggerTime Time   `json:"triggerTime"`
	Tag         string `json:"tag"`
	Ts          Time   `json:"ts"`
}

// CancelAllAfter is a dead man's switch: unless called again within timeout
// (10s to 120s), all pending orders are cancelled. Call it periodically from
// a bot; a timeout of 0 disarms it.
func (s *TradeService) CancelAllAfter(ctx context.Context, timeout time.Duration) (CancelAllAfterResult, error) {
	body := map[string]string{"timeOut": strconv.Itoa(int(timeout / time.Second))}
	return one[CancelAllAfterResult](ctx, s.c, privatePost("/api/v5/trade/cancel-all-after", body))
}

const clOrdIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// NewClientOrderID returns a random 32-character clOrdId. Generate it before
// sending an order and reuse it on retry so a duplicate cannot be placed.
func NewClientOrderID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = clOrdIDAlphabet[int(b[i])%len(clOrdIDAlphabet)]
	}
	return string(b)
}

func (r PlaceOrderRequest) instrument() string  { return r.InstID }
func (r CancelOrderRequest) instrument() string { return r.InstID }
func (r AmendOrderRequest) instrument() string  { return r.InstID }
