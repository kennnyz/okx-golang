package okx

import "context"

// PlaceAlgoOrderRequest places a conditional order. Which fields apply
// depends on OrdType:
//   - conditional, oco: Tp*/Sl* trigger and order prices
//   - trigger: TriggerPx, OrderPx, optionally AttachAlgoOrds
//   - move_order_stop: CallbackRatio or CallbackSpread, optionally ActivePx
//   - twap, iceberg: PxVar or PxSpread, SzLimit, PxLimit, TimeInterval
//   - chase: ChaseType, ChaseVal, MaxChaseType, MaxChaseVal
type PlaceAlgoOrderRequest struct {
	InstID        string      `json:"instId"`
	TdMode        TdMode      `json:"tdMode"`
	Side          Side        `json:"side"`
	OrdType       AlgoOrdType `json:"ordType"`
	Sz            Number      `json:"sz,omitempty"`
	CloseFraction Number      `json:"closeFraction,omitempty"`
	PosSide       PosSide     `json:"posSide,omitempty"`
	Ccy           string      `json:"ccy,omitempty"`
	Tag           string      `json:"tag,omitempty"`
	TgtCcy        TgtCcy      `json:"tgtCcy,omitempty"`
	AlgoClOrdID   string      `json:"algoClOrdId,omitempty"`
	ReduceOnly    bool        `json:"reduceOnly,omitempty"`
	CxlOnClosePos bool        `json:"cxlOnClosePos,omitempty"`
	TradeQuoteCcy string      `json:"tradeQuoteCcy,omitempty"`

	TpTriggerPx     Number        `json:"tpTriggerPx,omitempty"`
	TpTriggerPxType TriggerPxType `json:"tpTriggerPxType,omitempty"`
	TpOrdPx         Number        `json:"tpOrdPx,omitempty"`
	TpOrdKind       string        `json:"tpOrdKind,omitempty"`
	SlTriggerPx     Number        `json:"slTriggerPx,omitempty"`
	SlTriggerPxType TriggerPxType `json:"slTriggerPxType,omitempty"`
	SlOrdPx         Number        `json:"slOrdPx,omitempty"`

	TriggerPx      Number            `json:"triggerPx,omitempty"`
	TriggerPxType  TriggerPxType     `json:"triggerPxType,omitempty"`
	OrderPx        Number            `json:"orderPx,omitempty"`
	AdvanceOrdType string            `json:"advanceOrdType,omitempty"`
	AttachAlgoOrds []AttachAlgoOrder `json:"attachAlgoOrds,omitempty"`

	CallbackRatio  Number `json:"callbackRatio,omitempty"`
	CallbackSpread Number `json:"callbackSpread,omitempty"`
	ActivePx       Number `json:"activePx,omitempty"`

	PxVar        Number `json:"pxVar,omitempty"`
	PxSpread     Number `json:"pxSpread,omitempty"`
	SzLimit      Number `json:"szLimit,omitempty"`
	PxLimit      Number `json:"pxLimit,omitempty"`
	TimeInterval string `json:"timeInterval,omitempty"`

	ChaseType    string `json:"chaseType,omitempty"`
	ChaseVal     Number `json:"chaseVal,omitempty"`
	MaxChaseType string `json:"maxChaseType,omitempty"`
	MaxChaseVal  Number `json:"maxChaseVal,omitempty"`
}

type AlgoOrderResult struct {
	AlgoID      string `json:"algoId"`
	AlgoClOrdID string `json:"algoClOrdId"`
	ClOrdID     string `json:"clOrdId"`
	ReqID       string `json:"reqId"`
	Tag         string `json:"tag"`
	SCode       string `json:"sCode"`
	SMsg        string `json:"sMsg"`
}

// Err returns the item's error, or nil if it succeeded.
func (r AlgoOrderResult) Err() error {
	if r.SCode == "" || r.SCode == "0" {
		return nil
	}
	return &APIError{Code: r.SCode, Msg: r.SMsg}
}

// PlaceAlgoOrder places a TP/SL, trigger, trailing stop, TWAP, iceberg or
// chase order.
func (s *TradeService) PlaceAlgoOrder(ctx context.Context, req PlaceAlgoOrderRequest) (AlgoOrderResult, error) {
	return one[AlgoOrderResult](ctx, s.c, tradePost("/api/v5/trade/order-algo", req, req.InstID))
}

type CancelAlgoOrderRequest struct {
	InstID      string `json:"instId"`
	AlgoID      string `json:"algoId,omitempty"`
	AlgoClOrdID string `json:"algoClOrdId,omitempty"`
}

// CancelAlgoOrders cancels up to 10 algo orders; check Err on each result.
func (s *TradeService) CancelAlgoOrders(ctx context.Context, reqs []CancelAlgoOrderRequest) ([]AlgoOrderResult, error) {
	r := tradePost("/api/v5/trade/cancel-algos", reqs, firstInst(reqs, func(r CancelAlgoOrderRequest) string { return r.InstID }))
	r.batch = true
	return list[AlgoOrderResult](ctx, s.c, r)
}

type AmendAlgoOrderRequest struct {
	InstID             string        `json:"instId"`
	AlgoID             string        `json:"algoId,omitempty"`
	AlgoClOrdID        string        `json:"algoClOrdId,omitempty"`
	ReqID              string        `json:"reqId,omitempty"`
	CxlOnFail          bool          `json:"cxlOnFail,omitempty"`
	NewSz              Number        `json:"newSz,omitempty"`
	NewTpTriggerPx     Number        `json:"newTpTriggerPx,omitempty"`
	NewTpTriggerPxType TriggerPxType `json:"newTpTriggerPxType,omitempty"`
	NewTpOrdPx         Number        `json:"newTpOrdPx,omitempty"`
	NewSlTriggerPx     Number        `json:"newSlTriggerPx,omitempty"`
	NewSlTriggerPxType TriggerPxType `json:"newSlTriggerPxType,omitempty"`
	NewSlOrdPx         Number        `json:"newSlOrdPx,omitempty"`
	NewTriggerPx       Number        `json:"newTriggerPx,omitempty"`
	NewTriggerPxType   TriggerPxType `json:"newTriggerPxType,omitempty"`
	NewOrdPx           Number        `json:"newOrdPx,omitempty"`
}

// AmendAlgoOrder changes a pending TP/SL or trigger order.
func (s *TradeService) AmendAlgoOrder(ctx context.Context, req AmendAlgoOrderRequest) (AlgoOrderResult, error) {
	return one[AlgoOrderResult](ctx, s.c, tradePost("/api/v5/trade/amend-algos", req, req.InstID))
}

type AlgoOrder struct {
	InstType        InstType          `json:"instType"`
	InstID          string            `json:"instId"`
	AlgoID          string            `json:"algoId"`
	AlgoClOrdID     string            `json:"algoClOrdId"`
	ClOrdID         string            `json:"clOrdId"`
	OrdID           string            `json:"ordId"`
	OrdIDList       []string          `json:"ordIdList"`
	OrdType         AlgoOrdType       `json:"ordType"`
	State           AlgoState         `json:"state"`
	Side            Side              `json:"side"`
	PosSide         PosSide           `json:"posSide"`
	TdMode          TdMode            `json:"tdMode"`
	TgtCcy          TgtCcy            `json:"tgtCcy"`
	Ccy             string            `json:"ccy"`
	Sz              Number            `json:"sz"`
	CloseFraction   Number            `json:"closeFraction"`
	Lever           Number            `json:"lever"`
	TpTriggerPx     Number            `json:"tpTriggerPx"`
	TpTriggerPxType TriggerPxType     `json:"tpTriggerPxType"`
	TpOrdPx         Number            `json:"tpOrdPx"`
	SlTriggerPx     Number            `json:"slTriggerPx"`
	SlTriggerPxType TriggerPxType     `json:"slTriggerPxType"`
	SlOrdPx         Number            `json:"slOrdPx"`
	TriggerPx       Number            `json:"triggerPx"`
	TriggerPxType   TriggerPxType     `json:"triggerPxType"`
	OrdPx           Number            `json:"ordPx"`
	ActualSz        Number            `json:"actualSz"`
	ActualPx        Number            `json:"actualPx"`
	ActualSide      string            `json:"actualSide"`
	TriggerTime     Time              `json:"triggerTime"`
	CallbackRatio   Number            `json:"callbackRatio"`
	CallbackSpread  Number            `json:"callbackSpread"`
	ActivePx        Number            `json:"activePx"`
	MoveTriggerPx   Number            `json:"moveTriggerPx"`
	PxVar           Number            `json:"pxVar"`
	PxSpread        Number            `json:"pxSpread"`
	SzLimit         Number            `json:"szLimit"`
	PxLimit         Number            `json:"pxLimit"`
	TimeInterval    string            `json:"timeInterval"`
	ChaseType       string            `json:"chaseType"`
	ChaseVal        Number            `json:"chaseVal"`
	MaxChaseType    string            `json:"maxChaseType"`
	MaxChaseVal     Number            `json:"maxChaseVal"`
	ReduceOnly      Bool              `json:"reduceOnly"`
	Last            Number            `json:"last"`
	FailCode        string            `json:"failCode"`
	Tag             string            `json:"tag"`
	AttachAlgoOrds  []AttachAlgoOrder `json:"attachAlgoOrds"`
	CTime           Time              `json:"cTime"`
	UTime           Time              `json:"uTime"`
}

type AlgoOrderRequest struct {
	AlgoID      string `json:"algoId,omitempty"`
	AlgoClOrdID string `json:"algoClOrdId,omitempty"`
}

// AlgoOrder returns one algo order by algoId or algoClOrdId.
func (s *TradeService) AlgoOrder(ctx context.Context, req AlgoOrderRequest) (AlgoOrder, error) {
	return one[AlgoOrder](ctx, s.c, privateGet("/api/v5/trade/order-algo", req))
}

type AlgoOrdersRequest struct {
	OrdType     AlgoOrdType `json:"ordType"`
	State       AlgoState   `json:"state,omitempty"`
	AlgoID      string      `json:"algoId,omitempty"`
	AlgoClOrdID string      `json:"algoClOrdId,omitempty"`
	InstType    InstType    `json:"instType,omitempty"`
	InstID      string      `json:"instId,omitempty"`
	After       string      `json:"after,omitempty"`
	Before      string      `json:"before,omitempty"`
	Limit       int         `json:"limit,omitempty"`
}

// PendingAlgoOrders returns untriggered algo orders of one type.
func (s *TradeService) PendingAlgoOrders(ctx context.Context, req AlgoOrdersRequest) ([]AlgoOrder, error) {
	return list[AlgoOrder](ctx, s.c, privateGet("/api/v5/trade/orders-algo-pending", req))
}

// AlgoOrdersHistory returns finished algo orders of the last 3 months. Set
// either State or AlgoID.
func (s *TradeService) AlgoOrdersHistory(ctx context.Context, req AlgoOrdersRequest) ([]AlgoOrder, error) {
	return list[AlgoOrder](ctx, s.c, privateGet("/api/v5/trade/orders-algo-history", req))
}
