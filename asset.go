package okx

import "context"

// AssetService covers /api/v5/asset: the funding account, transfers,
// deposits and withdrawals.
type AssetService struct{ c *Client }

type Currency struct {
	Ccy                  string `json:"ccy"`
	Name                 string `json:"name"`
	Chain                string `json:"chain"`
	CtAddr               string `json:"ctAddr"`
	LogoLink             string `json:"logoLink"`
	CanDep               bool   `json:"canDep"`
	CanWd                bool   `json:"canWd"`
	CanInternal          bool   `json:"canInternal"`
	MainNet              bool   `json:"mainNet"`
	NeedTag              bool   `json:"needTag"`
	MinDep               Number `json:"minDep"`
	MinWd                Number `json:"minWd"`
	MaxWd                Number `json:"maxWd"`
	WdTickSz             Number `json:"wdTickSz"`
	WdQuota              Number `json:"wdQuota"`
	UsedWdQuota          Number `json:"usedWdQuota"`
	Fee                  Number `json:"fee"`
	MinFee               Number `json:"minFee"`
	MaxFee               Number `json:"maxFee"`
	MinDepArrivalConfirm Number `json:"minDepArrivalConfirm"`
	MinWdUnlockConfirm   Number `json:"minWdUnlockConfirm"`
	DepEstOpenTime       Time   `json:"depEstOpenTime"`
	WdEstOpenTime        Time   `json:"wdEstOpenTime"`
}

// Currencies returns deposit and withdrawal settings per currency and chain.
func (s *AssetService) Currencies(ctx context.Context, ccys ...string) ([]Currency, error) {
	return list[Currency](ctx, s.c, privateGet("/api/v5/asset/currencies", ccyQuery(ccys)))
}

type FundingBalance struct {
	Ccy       string `json:"ccy"`
	Bal       Number `json:"bal"`
	AvailBal  Number `json:"availBal"`
	FrozenBal Number `json:"frozenBal"`
}

// Balances returns the funding account balances.
func (s *AssetService) Balances(ctx context.Context, ccys ...string) ([]FundingBalance, error) {
	return list[FundingBalance](ctx, s.c, privateGet("/api/v5/asset/balances", ccyQuery(ccys)))
}

type TransferRequest struct {
	Ccy  string  `json:"ccy"`
	Amt  Number  `json:"amt"`
	From Account `json:"from"`
	To   Account `json:"to"`
	// Type: "0" within account (default), "1" master to sub, "2" sub to
	// master, "3" sub to master by sub key, "4" sub to sub.
	Type      string `json:"type,omitempty"`
	SubAcct   string `json:"subAcct,omitempty"`
	LoanTrans bool   `json:"loanTrans,omitempty"`
	ClientID  string `json:"clientId,omitempty"`
}

type Transfer struct {
	TransID  string  `json:"transId"`
	ClientID string  `json:"clientId"`
	Ccy      string  `json:"ccy"`
	Amt      Number  `json:"amt"`
	From     Account `json:"from"`
	To       Account `json:"to"`
	State    string  `json:"state"`
	Type     string  `json:"type"`
	SubAcct  string  `json:"subAcct"`
}

// Transfer moves funds between the funding and trading accounts or
// sub-accounts.
func (s *AssetService) Transfer(ctx context.Context, req TransferRequest) (Transfer, error) {
	return one[Transfer](ctx, s.c, privatePost("/api/v5/asset/transfer", req))
}

// TransferState returns the state of a transfer: "success", "pending" or
// "failed".
func (s *AssetService) TransferState(ctx context.Context, transID string) (Transfer, error) {
	return one[Transfer](ctx, s.c, privateGet("/api/v5/asset/transfer-state", map[string]string{"transId": transID}))
}

type DepositAddress struct {
	Ccy      string `json:"ccy"`
	Chain    string `json:"chain"`
	Addr     string `json:"addr"`
	Tag      string `json:"tag"`
	Memo     string `json:"memo"`
	PmtID    string `json:"pmtId"`
	CtAddr   string `json:"ctAddr"`
	To       string `json:"to"`
	Selected bool   `json:"selected"`
}

// DepositAddresses returns deposit addresses of a currency on every chain.
func (s *AssetService) DepositAddresses(ctx context.Context, ccy string) ([]DepositAddress, error) {
	return list[DepositAddress](ctx, s.c, privateGet("/api/v5/asset/deposit-address", map[string]string{"ccy": ccy}))
}

type Deposit struct {
	DepID               string `json:"depId"`
	Ccy                 string `json:"ccy"`
	Chain               string `json:"chain"`
	Amt                 Number `json:"amt"`
	From                string `json:"from"`
	To                  string `json:"to"`
	TxID                string `json:"txId"`
	FromWdID            string `json:"fromWdId"`
	State               string `json:"state"`
	ActualDepBlkConfirm string `json:"actualDepBlkConfirm"`
	Ts                  Time   `json:"ts"`
}

type DepositHistoryRequest struct {
	Ccy      string `json:"ccy,omitempty"`
	DepID    string `json:"depId,omitempty"`
	FromWdID string `json:"fromWdId,omitempty"`
	TxID     string `json:"txId,omitempty"`
	Type     string `json:"type,omitempty"`
	State    string `json:"state,omitempty"`
	After    Time   `json:"after,omitempty"`
	Before   Time   `json:"before,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func (s *AssetService) DepositHistory(ctx context.Context, req DepositHistoryRequest) ([]Deposit, error) {
	return list[Deposit](ctx, s.c, privateGet("/api/v5/asset/deposit-history", req))
}

type WithdrawRequest struct {
	Ccy string `json:"ccy"`
	Amt Number `json:"amt"`
	// Dest is "3" for an internal OKX transfer or "4" for on-chain.
	Dest       string `json:"dest"`
	ToAddr     string `json:"toAddr"`
	ToAddrType string `json:"toAddrType,omitempty"`
	Chain      string `json:"chain,omitempty"`
	AreaCode   string `json:"areaCode,omitempty"`
	ClientID   string `json:"clientId,omitempty"`
	RcvrInfo   any    `json:"rcvrInfo,omitempty"`
}

type Withdrawal struct {
	WdID     string `json:"wdId"`
	ClientID string `json:"clientId"`
	Ccy      string `json:"ccy"`
	Chain    string `json:"chain"`
	Amt      Number `json:"amt"`
	Fee      Number `json:"fee"`
	FeeCcy   string `json:"feeCcy"`
	From     string `json:"from"`
	To       string `json:"to"`
	Tag      string `json:"tag"`
	Memo     string `json:"memo"`
	PmtID    string `json:"pmtId"`
	TxID     string `json:"txId"`
	State    string `json:"state"`
	Ts       Time   `json:"ts"`
}

// Withdraw sends funds to an address. The destination must be in the
// account's withdrawal address whitelist when using API keys.
func (s *AssetService) Withdraw(ctx context.Context, req WithdrawRequest) (Withdrawal, error) {
	return one[Withdrawal](ctx, s.c, privatePost("/api/v5/asset/withdrawal", req))
}

// CancelWithdrawal cancels a withdrawal that has not been broadcast yet.
func (s *AssetService) CancelWithdrawal(ctx context.Context, wdID string) error {
	_, err := one[Withdrawal](ctx, s.c, privatePost("/api/v5/asset/cancel-withdrawal", map[string]string{"wdId": wdID}))
	return err
}

type WithdrawalHistoryRequest struct {
	Ccy      string `json:"ccy,omitempty"`
	WdID     string `json:"wdId,omitempty"`
	ClientID string `json:"clientId,omitempty"`
	TxID     string `json:"txId,omitempty"`
	Type     string `json:"type,omitempty"`
	State    string `json:"state,omitempty"`
	After    Time   `json:"after,omitempty"`
	Before   Time   `json:"before,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func (s *AssetService) WithdrawalHistory(ctx context.Context, req WithdrawalHistoryRequest) ([]Withdrawal, error) {
	return list[Withdrawal](ctx, s.c, privateGet("/api/v5/asset/withdrawal-history", req))
}

type AssetBill struct {
	BillID   string `json:"billId"`
	ClientID string `json:"clientId"`
	Ccy      string `json:"ccy"`
	BalChg   Number `json:"balChg"`
	Bal      Number `json:"bal"`
	Type     string `json:"type"`
	Notes    string `json:"notes"`
	Ts       Time   `json:"ts"`
}

type AssetBillsRequest struct {
	Ccy      string `json:"ccy,omitempty"`
	Type     string `json:"type,omitempty"`
	ClientID string `json:"clientId,omitempty"`
	After    Time   `json:"after,omitempty"`
	Before   Time   `json:"before,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// Bills returns funding account balance changes of the last month.
func (s *AssetService) Bills(ctx context.Context, req AssetBillsRequest) ([]AssetBill, error) {
	return list[AssetBill](ctx, s.c, privateGet("/api/v5/asset/bills", req))
}
