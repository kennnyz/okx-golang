//go:build integration

package okx_test

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	okx "github.com/kennnyz/okx-golang"
)

func liveCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestLivePublicREST(t *testing.T) {
	ctx := liveCtx(t)
	c := okx.NewClient()

	tk, err := c.Market.Ticker(ctx, "BTC-USDT")
	if err != nil || tk.Last.IsZero() || tk.Ts.IsZero() {
		t.Fatalf("ticker: %+v, %v", tk, err)
	}
	book, err := c.Market.OrderBook(ctx, "BTC-USDT", 5)
	if err != nil || len(book.Asks) != 5 || book.SeqID == 0 {
		t.Fatalf("book: %+v, %v", book, err)
	}
	candles, err := c.Market.Candles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar1H, Limit: 3})
	if err != nil || len(candles) != 3 || candles[0].Close.IsZero() {
		t.Fatalf("candles: %+v, %v", candles, err)
	}
	idx, err := c.Market.IndexCandles(ctx, okx.CandlesRequest{InstID: "BTC-USD", Bar: okx.Bar1H, Limit: 2})
	if err != nil || len(idx) != 2 || !idx[1].Confirmed {
		t.Fatalf("index candles: %+v, %v", idx, err)
	}
	n := 0
	for c, err := range c.Market.AllHistoryCandles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar1D, Limit: 100}) {
		if err != nil {
			t.Fatal(err)
		}
		if c.Open.IsZero() {
			t.Fatalf("empty candle %+v", c)
		}
		if n++; n == 250 {
			break
		}
	}
	if _, err := c.Market.Trades(ctx, "BTC-USDT", 10); err != nil {
		t.Fatal(err)
	}
	insts, err := c.Public.Instruments(ctx, okx.InstrumentsRequest{InstType: okx.InstSwap, InstID: "BTC-USDT-SWAP"})
	if err != nil || len(insts) != 1 || insts[0].CtVal.IsZero() || insts[0].ListTime.IsZero() {
		t.Fatalf("instruments: %+v, %v", insts, err)
	}
	fr, err := c.Public.FundingRate(ctx, "BTC-USDT-SWAP")
	if err != nil || fr.FundingTime.IsZero() {
		t.Fatalf("funding: %+v, %v", fr, err)
	}
	liq, err := c.Public.LiquidationOrders(ctx, okx.LiquidationOrdersRequest{InstType: okx.InstSwap, InstFamily: "BTC-USDT", State: "filled", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(liq) > 0 && len(liq[0].Details) > 0 && liq[0].Details[0].Ts.IsZero() {
		t.Fatalf("liquidation: %+v", liq)
	}
	flow, err := c.Public.TakerVolume(ctx, okx.TakerVolumeRequest{Ccy: "BTC", InstType: okx.InstSpot, Period: "1H"})
	if err != nil || len(flow) == 0 || flow[0].BuyVol.IsZero() {
		t.Fatalf("taker volume: %+v, %v", flow, err)
	}
	if err := c.SyncTime(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLiveStream(t *testing.T) {
	ctx := liveCtx(t)
	s := okx.NewStream()
	defer func() { _ = s.Close() }()

	tickers, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	if tk := <-tickers.C; tk.InstID != "BTC-USDT" || tk.Last.IsZero() {
		t.Fatalf("ticker push: %+v", tk)
	}

	books, err := s.OrderBook(ctx, "BTC-USDT", okx.Books)
	if err != nil {
		t.Fatal(err)
	}
	var book okx.Book
	for i := 0; i < 20; i++ {
		u := <-books.C
		if i == 0 && !u.IsSnapshot() {
			t.Fatalf("first push is %q", u.Action)
		}
		if err := book.Apply(u); err != nil {
			t.Fatalf("apply update %d: %v", i, err)
		}
	}
	bid, _ := book.BestBid()
	ask, _ := book.BestAsk()
	if bid.Px.Float64() >= ask.Px.Float64() || len(book.Asks) < 100 {
		t.Fatalf("crossed or shallow book: bid %v ask %v depth %d", bid.Px, ask.Px, len(book.Asks))
	}

	candles, err := s.Candles(ctx, "BTC-USDT", okx.Bar1m)
	if err != nil {
		t.Fatal(err)
	}
	if c := <-candles.C; c.Close.IsZero() {
		t.Fatalf("candle push: %+v", c)
	}

	if err := tickers.Unsubscribe(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-tickers.C; ok {
		for range tickers.C {
		}
	}
}

func TestLiveDemoPrivate(t *testing.T) {
	key, secret, pass := os.Getenv("OKX_DEMO_API_KEY"), os.Getenv("OKX_DEMO_SECRET_KEY"), os.Getenv("OKX_DEMO_PASSPHRASE")
	if key == "" {
		t.Skip("OKX_DEMO_API_KEY not set")
	}
	ctx := liveCtx(t)
	opts := []okx.Option{okx.WithCredentials(key, secret, pass), okx.WithDemoTrading()}
	c := okx.NewClient(opts...)

	if _, err := c.Account.Balance(ctx); err != nil {
		t.Fatal(err)
	}
	cfg, err := c.Account.Config(ctx)
	if err != nil || cfg.UID == "" {
		t.Fatalf("config: %+v, %v", cfg, err)
	}

	s := okx.NewStream(opts...)
	defer func() { _ = s.Close() }()
	orders, err := s.Orders(ctx, okx.InstSpot, "")
	if err != nil {
		t.Fatal(err)
	}
	tk, err := c.Market.Ticker(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	px := okx.NumberFromFloat(math.Round(tk.Last.Float64() / 2))
	res, err := s.PlaceOrder(ctx, okx.PlaceOrderRequest{
		InstID: "BTC-USDT", TdMode: okx.TdCash, Side: okx.SideBuy, OrdType: okx.OrdLimit,
		Px: px, Sz: "0.0001", ClOrdID: okx.NewClientOrderID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if o := <-orders.C; o.OrdID != res.OrdID || o.State != okx.StateLive {
		t.Fatalf("order push: %+v", o)
	}
	if _, err := c.Trade.CancelOrder(ctx, okx.CancelOrderRequest{InstID: "BTC-USDT", OrdID: res.OrdID}); err != nil {
		t.Fatal(err)
	}
}
