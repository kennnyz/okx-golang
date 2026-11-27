package okx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestEndpoints checks that every wrapped endpoint hits the method and path
// from the OKX v5 docs, signs private calls, sends GET params as a query and
// POST params as a JSON body, and decodes a minimal response.
func TestEndpoints(t *testing.T) {
	ctx := context.Background()
	candleRow := `[["1700000000000","1","2","0.5","1.5","10","20","20","1"]]`

	tests := []struct {
		method, path string
		private      bool
		data         string // response "data"; defaults to [{}]
		call         func(c *Client) error
	}{
		// Market
		{"GET", "/api/v5/market/tickers", false, "", func(c *Client) error {
			_, err := c.Market.Tickers(ctx, TickersRequest{InstType: InstSpot})
			return err
		}},
		{"GET", "/api/v5/market/ticker", false, "", func(c *Client) error {
			_, err := c.Market.Ticker(ctx, "BTC-USDT")
			return err
		}},
		{"GET", "/api/v5/market/index-tickers", false, "", func(c *Client) error {
			_, err := c.Market.IndexTickers(ctx, IndexTickersRequest{InstID: "BTC-USD"})
			return err
		}},
		{"GET", "/api/v5/market/books", false, "", func(c *Client) error {
			_, err := c.Market.OrderBook(ctx, "BTC-USDT", 5)
			return err
		}},
		{"GET", "/api/v5/market/books-full", false, "", func(c *Client) error {
			_, err := c.Market.FullOrderBook(ctx, "BTC-USDT", 100)
			return err
		}},
		{"GET", "/api/v5/market/history-candles", false, candleRow, func(c *Client) error {
			_, err := c.Market.HistoryCandles(ctx, CandlesRequest{InstID: "BTC-USDT"})
			return err
		}},
		{"GET", "/api/v5/market/index-candles", false, candleRow, func(c *Client) error {
			_, err := c.Market.IndexCandles(ctx, CandlesRequest{InstID: "BTC-USD"})
			return err
		}},
		{"GET", "/api/v5/market/history-index-candles", false, candleRow, func(c *Client) error {
			_, err := c.Market.HistoryIndexCandles(ctx, CandlesRequest{InstID: "BTC-USD"})
			return err
		}},
		{"GET", "/api/v5/market/mark-price-candles", false, candleRow, func(c *Client) error {
			_, err := c.Market.MarkPriceCandles(ctx, CandlesRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/market/history-mark-price-candles", false, candleRow, func(c *Client) error {
			_, err := c.Market.HistoryMarkPriceCandles(ctx, CandlesRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/market/trades", false, "", func(c *Client) error {
			_, err := c.Market.Trades(ctx, "BTC-USDT", 10)
			return err
		}},
		{"GET", "/api/v5/market/history-trades", false, "", func(c *Client) error {
			_, err := c.Market.HistoryTrades(ctx, HistoryTradesRequest{InstID: "BTC-USDT"})
			return err
		}},
		{"GET", "/api/v5/market/platform-24-volume", false, "", func(c *Client) error {
			_, err := c.Market.PlatformVolume24h(ctx)
			return err
		}},
		{"GET", "/api/v5/market/call-auction-details", false, "", func(c *Client) error {
			_, err := c.Market.CallAuction(ctx, "BTC-USDT")
			return err
		}},

		// Public data
		{"GET", "/api/v5/public/instruments", false, "", func(c *Client) error {
			_, err := c.Public.Instruments(ctx, InstrumentsRequest{InstType: InstSpot})
			return err
		}},
		{"GET", "/api/v5/public/funding-rate", false, "", func(c *Client) error {
			_, err := c.Public.FundingRate(ctx, "BTC-USDT-SWAP")
			return err
		}},
		{"GET", "/api/v5/public/funding-rate-history", false, "", func(c *Client) error {
			_, err := c.Public.FundingRateHistory(ctx, FundingRateHistoryRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/public/open-interest", false, "", func(c *Client) error {
			_, err := c.Public.OpenInterest(ctx, OpenInterestRequest{InstType: InstSwap})
			return err
		}},
		{"GET", "/api/v5/public/mark-price", false, "", func(c *Client) error {
			_, err := c.Public.MarkPrice(ctx, MarkPriceRequest{InstType: InstSwap})
			return err
		}},
		{"GET", "/api/v5/public/price-limit", false, "", func(c *Client) error {
			_, err := c.Public.PriceLimit(ctx, "BTC-USDT-SWAP")
			return err
		}},
		{"GET", "/api/v5/public/estimated-price", false, "", func(c *Client) error {
			_, err := c.Public.EstimatedPrice(ctx, "BTC-USD-251226")
			return err
		}},
		{"GET", "/api/v5/public/liquidation-orders", false, "", func(c *Client) error {
			_, err := c.Public.LiquidationOrders(ctx, LiquidationOrdersRequest{InstType: InstSwap})
			return err
		}},
		{"GET", "/api/v5/public/position-tiers", false, "", func(c *Client) error {
			_, err := c.Public.PositionTiers(ctx, PositionTiersRequest{InstType: InstSwap})
			return err
		}},
		{"GET", "/api/v5/public/convert-contract-coin", false, "", func(c *Client) error {
			_, err := c.Public.ConvertContractCoin(ctx, ConvertContractCoinRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/public/opt-summary", false, "", func(c *Client) error {
			_, err := c.Public.OptionSummary(ctx, OptionSummaryRequest{InstFamily: "BTC-USD"})
			return err
		}},
		{"GET", "/api/v5/system/status", false, "", func(c *Client) error {
			_, err := c.Public.SystemStatus(ctx)
			return err
		}},
		{"GET", "/api/v5/rubik/stat/taker-volume", false, `[["1700000000000","1","2"]]`, func(c *Client) error {
			_, err := c.Public.TakerVolume(ctx, TakerVolumeRequest{Ccy: "BTC"})
			return err
		}},
		{"GET", "/api/v5/rubik/stat/contracts/long-short-account-ratio", false, `[["1700000000000","1.5"]]`, func(c *Client) error {
			_, err := c.Public.LongShortAccountRatio(ctx, LongShortRatioRequest{Ccy: "BTC"})
			return err
		}},

		// Account
		{"GET", "/api/v5/account/positions", true, "", func(c *Client) error {
			_, err := c.Account.Positions(ctx, PositionsRequest{})
			return err
		}},
		{"GET", "/api/v5/account/positions-history", true, "", func(c *Client) error {
			_, err := c.Account.PositionsHistory(ctx, PositionsHistoryRequest{})
			return err
		}},
		{"GET", "/api/v5/account/config", true, "", func(c *Client) error {
			_, err := c.Account.Config(ctx)
			return err
		}},
		{"POST", "/api/v5/account/set-leverage", true, "", func(c *Client) error {
			_, err := c.Account.SetLeverage(ctx, SetLeverageRequest{InstID: "BTC-USDT-SWAP", Lever: "1"})
			return err
		}},
		{"GET", "/api/v5/account/leverage-info", true, "", func(c *Client) error {
			_, err := c.Account.LeverageInfo(ctx, LeverageInfoRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"POST", "/api/v5/account/set-position-mode", true, "", func(c *Client) error {
			return c.Account.SetPositionMode(ctx, PosModeNet)
		}},
		{"GET", "/api/v5/account/max-size", true, "", func(c *Client) error {
			_, err := c.Account.MaxOrderSize(ctx, MaxSizeRequest{InstID: "BTC-USDT"})
			return err
		}},
		{"GET", "/api/v5/account/max-avail-size", true, "", func(c *Client) error {
			_, err := c.Account.MaxAvailSize(ctx, MaxAvailSizeRequest{InstID: "BTC-USDT"})
			return err
		}},
		{"GET", "/api/v5/account/trade-fee", true, "", func(c *Client) error {
			_, err := c.Account.TradeFee(ctx, TradeFeeRequest{InstType: InstSpot})
			return err
		}},
		{"GET", "/api/v5/account/bills", true, "", func(c *Client) error {
			_, err := c.Account.Bills(ctx, BillsRequest{})
			return err
		}},
		{"GET", "/api/v5/account/bills-archive", true, "", func(c *Client) error {
			_, err := c.Account.BillsArchive(ctx, BillsRequest{})
			return err
		}},
		{"POST", "/api/v5/account/position/margin-balance", true, "", func(c *Client) error {
			_, err := c.Account.AdjustMargin(ctx, AdjustMarginRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/account/max-withdrawal", true, "", func(c *Client) error {
			_, err := c.Account.MaxWithdrawal(ctx, "USDT")
			return err
		}},

		// Funding account
		{"GET", "/api/v5/asset/currencies", true, "", func(c *Client) error {
			_, err := c.Asset.Currencies(ctx, "USDT")
			return err
		}},
		{"GET", "/api/v5/asset/balances", true, "", func(c *Client) error {
			_, err := c.Asset.Balances(ctx, "USDT")
			return err
		}},
		{"POST", "/api/v5/asset/transfer", true, "", func(c *Client) error {
			_, err := c.Asset.Transfer(ctx, TransferRequest{Ccy: "USDT", Amt: "1"})
			return err
		}},
		{"GET", "/api/v5/asset/transfer-state", true, "", func(c *Client) error {
			_, err := c.Asset.TransferState(ctx, "1")
			return err
		}},
		{"GET", "/api/v5/asset/deposit-address", true, "", func(c *Client) error {
			_, err := c.Asset.DepositAddresses(ctx, "USDT")
			return err
		}},
		{"GET", "/api/v5/asset/deposit-history", true, "", func(c *Client) error {
			_, err := c.Asset.DepositHistory(ctx, DepositHistoryRequest{})
			return err
		}},
		{"POST", "/api/v5/asset/withdrawal", true, "", func(c *Client) error {
			_, err := c.Asset.Withdraw(ctx, WithdrawRequest{Ccy: "USDT", Amt: "1"})
			return err
		}},
		{"POST", "/api/v5/asset/cancel-withdrawal", true, "", func(c *Client) error {
			return c.Asset.CancelWithdrawal(ctx, "1")
		}},
		{"GET", "/api/v5/asset/withdrawal-history", true, "", func(c *Client) error {
			_, err := c.Asset.WithdrawalHistory(ctx, WithdrawalHistoryRequest{})
			return err
		}},
		{"GET", "/api/v5/asset/bills", true, "", func(c *Client) error {
			_, err := c.Asset.Bills(ctx, AssetBillsRequest{})
			return err
		}},

		// Trade
		{"POST", "/api/v5/trade/batch-orders", true, "", func(c *Client) error {
			_, err := c.Trade.PlaceOrders(ctx, []PlaceOrderRequest{{InstID: "BTC-USDT"}})
			return err
		}},
		{"POST", "/api/v5/trade/cancel-order", true, "", func(c *Client) error {
			_, err := c.Trade.CancelOrder(ctx, CancelOrderRequest{InstID: "BTC-USDT", OrdID: "1"})
			return err
		}},
		{"POST", "/api/v5/trade/cancel-batch-orders", true, "", func(c *Client) error {
			_, err := c.Trade.CancelOrders(ctx, []CancelOrderRequest{{InstID: "BTC-USDT", OrdID: "1"}})
			return err
		}},
		{"POST", "/api/v5/trade/amend-order", true, "", func(c *Client) error {
			_, err := c.Trade.AmendOrder(ctx, AmendOrderRequest{InstID: "BTC-USDT", OrdID: "1"})
			return err
		}},
		{"POST", "/api/v5/trade/amend-batch-orders", true, "", func(c *Client) error {
			_, err := c.Trade.AmendOrders(ctx, []AmendOrderRequest{{InstID: "BTC-USDT", OrdID: "1"}})
			return err
		}},
		{"POST", "/api/v5/trade/close-position", true, "", func(c *Client) error {
			_, err := c.Trade.ClosePosition(ctx, ClosePositionRequest{InstID: "BTC-USDT-SWAP"})
			return err
		}},
		{"GET", "/api/v5/trade/order", true, "", func(c *Client) error {
			_, err := c.Trade.Order(ctx, OrderRequest{InstID: "BTC-USDT", OrdID: "1"})
			return err
		}},
		{"GET", "/api/v5/trade/orders-pending", true, "", func(c *Client) error {
			_, err := c.Trade.PendingOrders(ctx, PendingOrdersRequest{})
			return err
		}},
		{"GET", "/api/v5/trade/orders-history", true, "", func(c *Client) error {
			_, err := c.Trade.OrdersHistory(ctx, OrdersHistoryRequest{InstType: InstSpot})
			return err
		}},
		{"GET", "/api/v5/trade/orders-history-archive", true, "", func(c *Client) error {
			_, err := c.Trade.OrdersHistoryArchive(ctx, OrdersHistoryRequest{InstType: InstSpot})
			return err
		}},
		{"GET", "/api/v5/trade/fills", true, "", func(c *Client) error {
			_, err := c.Trade.Fills(ctx, FillsRequest{})
			return err
		}},
		{"GET", "/api/v5/trade/fills-history", true, "", func(c *Client) error {
			_, err := c.Trade.FillsHistory(ctx, FillsRequest{InstType: InstSpot})
			return err
		}},
		{"POST", "/api/v5/trade/cancel-all-after", true, "", func(c *Client) error {
			_, err := c.Trade.CancelAllAfter(ctx, 10*time.Second)
			return err
		}},

		// Algo orders
		{"POST", "/api/v5/trade/order-algo", true, "", func(c *Client) error {
			_, err := c.Trade.PlaceAlgoOrder(ctx, PlaceAlgoOrderRequest{InstID: "BTC-USDT"})
			return err
		}},
		{"POST", "/api/v5/trade/cancel-algos", true, "", func(c *Client) error {
			_, err := c.Trade.CancelAlgoOrders(ctx, []CancelAlgoOrderRequest{{InstID: "BTC-USDT", AlgoID: "1"}})
			return err
		}},
		{"POST", "/api/v5/trade/amend-algos", true, "", func(c *Client) error {
			_, err := c.Trade.AmendAlgoOrder(ctx, AmendAlgoOrderRequest{InstID: "BTC-USDT", AlgoID: "1"})
			return err
		}},
		{"GET", "/api/v5/trade/order-algo", true, "", func(c *Client) error {
			_, err := c.Trade.AlgoOrder(ctx, AlgoOrderRequest{AlgoID: "1"})
			return err
		}},
		{"GET", "/api/v5/trade/orders-algo-pending", true, "", func(c *Client) error {
			_, err := c.Trade.PendingAlgoOrders(ctx, AlgoOrdersRequest{OrdType: AlgoConditional})
			return err
		}},
		{"GET", "/api/v5/trade/orders-algo-history", true, "", func(c *Client) error {
			_, err := c.Trade.AlgoOrdersHistory(ctx, AlgoOrdersRequest{OrdType: AlgoConditional})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			hit := false
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				hit = true
				if r.Method != tt.method || r.URL.Path != tt.path {
					t.Errorf("request %s %s", r.Method, r.URL.Path)
				}
				if signed := r.Header.Get("OK-ACCESS-SIGN") != ""; signed != tt.private {
					t.Errorf("signed = %v, want %v", signed, tt.private)
				}
				body, _ := io.ReadAll(r.Body)
				if r.Method == http.MethodPost && !json.Valid(body) {
					t.Errorf("body is not JSON: %q", body)
				}
				if r.Method == http.MethodGet && len(body) != 0 {
					t.Errorf("GET with body %q", body)
				}
				data := tt.data
				if data == "" {
					data = `[{}]`
				}
				_, _ = io.Copy(w, strings.NewReader(`{"code":"0","msg":"","data":`+data+`}`))
			}, WithCredentials("k", "s", "p"))

			if err := tt.call(c); err != nil {
				t.Fatal(err)
			}
			if !hit {
				t.Fatal("no request sent")
			}
		})
	}
}
