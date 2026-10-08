package okx

type InstType string

const (
	InstSpot    InstType = "SPOT"
	InstMargin  InstType = "MARGIN"
	InstSwap    InstType = "SWAP"
	InstFutures InstType = "FUTURES"
	InstOption  InstType = "OPTION"
	InstEvents  InstType = "EVENTS"
	InstAny     InstType = "ANY"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type PosSide string

const (
	PosSideLong  PosSide = "long"
	PosSideShort PosSide = "short"
	PosSideNet   PosSide = "net"
)

type OrdType string

const (
	OrdMarket          OrdType = "market"
	OrdLimit           OrdType = "limit"
	OrdPostOnly        OrdType = "post_only"
	OrdFOK             OrdType = "fok"
	OrdIOC             OrdType = "ioc"
	OrdOptimalLimitIOC OrdType = "optimal_limit_ioc"
	OrdMMP             OrdType = "mmp"
	OrdMMPAndPostOnly  OrdType = "mmp_and_post_only"
	OrdRPI             OrdType = "rpi"
)

// TdMode is the trade mode of an order.
type TdMode string

const (
	TdCash     TdMode = "cash"
	TdCross    TdMode = "cross"
	TdIsolated TdMode = "isolated"
	TdSpotIso  TdMode = "spot_isolated"
)

type MgnMode string

const (
	MgnCross    MgnMode = "cross"
	MgnIsolated MgnMode = "isolated"
)

type OrderState string

const (
	StateLive            OrderState = "live"
	StatePartiallyFilled OrderState = "partially_filled"
	StateFilled          OrderState = "filled"
	StateCanceled        OrderState = "canceled"
	StateMMPCanceled     OrderState = "mmp_canceled"
)

// TgtCcy selects the unit of sz for spot market orders.
type TgtCcy string

const (
	TgtBaseCcy  TgtCcy = "base_ccy"
	TgtQuoteCcy TgtCcy = "quote_ccy"
)

type PosMode string

const (
	PosModeLongShort PosMode = "long_short_mode"
	PosModeNet       PosMode = "net_mode"
)

type AlgoOrdType string

const (
	AlgoConditional AlgoOrdType = "conditional"
	AlgoOCO         AlgoOrdType = "oco"
	AlgoTrigger     AlgoOrdType = "trigger"
	AlgoTrailing    AlgoOrdType = "move_order_stop"
	AlgoTWAP        AlgoOrdType = "twap"
	AlgoIceberg     AlgoOrdType = "iceberg"
	AlgoChase       AlgoOrdType = "chase"
)

type AlgoState string

const (
	AlgoStateLive          AlgoState = "live"
	AlgoStatePause         AlgoState = "pause"
	AlgoStatePartiallyEff  AlgoState = "partially_effective"
	AlgoStateEffective     AlgoState = "effective"
	AlgoStateCanceled      AlgoState = "canceled"
	AlgoStateOrderFailed   AlgoState = "order_failed"
	AlgoStatePartiallyFail AlgoState = "partially_failed"
)

// TriggerPxType is the price used to evaluate a trigger.
type TriggerPxType string

const (
	TriggerLast  TriggerPxType = "last"
	TriggerIndex TriggerPxType = "index"
	TriggerMark  TriggerPxType = "mark"
)

// Bar is a candlestick interval. Uppercase H/D/W/M bars are in Hong Kong
// time; the "utc" variants align to UTC.
type Bar string

const (
	Bar1s     Bar = "1s"
	Bar1m     Bar = "1m"
	Bar3m     Bar = "3m"
	Bar5m     Bar = "5m"
	Bar15m    Bar = "15m"
	Bar30m    Bar = "30m"
	Bar1H     Bar = "1H"
	Bar2H     Bar = "2H"
	Bar4H     Bar = "4H"
	Bar6H     Bar = "6H"
	Bar12H    Bar = "12H"
	Bar1D     Bar = "1D"
	Bar2D     Bar = "2D"
	Bar3D     Bar = "3D"
	Bar1W     Bar = "1W"
	Bar1M     Bar = "1M"
	Bar3M     Bar = "3M"
	Bar6Hutc  Bar = "6Hutc"
	Bar12Hutc Bar = "12Hutc"
	Bar1Dutc  Bar = "1Dutc"
	Bar2Dutc  Bar = "2Dutc"
	Bar3Dutc  Bar = "3Dutc"
	Bar1Wutc  Bar = "1Wutc"
	Bar1Mutc  Bar = "1Mutc"
	Bar3Mutc  Bar = "3Mutc"
)

// Account is an OKX account type used in funds transfers.
type Account string

const (
	AccountFunding Account = "6"
	AccountTrading Account = "18"
)
