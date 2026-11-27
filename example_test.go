package okx_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	okx "github.com/kennnyz/okx-golang"
)

func ExampleNewClient() {
	client := okx.NewClient()
	ticker, err := client.Market.Ticker(context.Background(), "BTC-USDT")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ticker.Last, ticker.Ts.Format(time.RFC3339))
}

func ExampleTradeService_PlaceOrder() {
	client := okx.NewClient(okx.WithCredentials("key", "secret", "passphrase"), okx.WithDemoTrading())
	res, err := client.Trade.PlaceOrder(context.Background(), okx.PlaceOrderRequest{
		InstID:  "BTC-USDT",
		TdMode:  okx.TdCash,
		Side:    okx.SideBuy,
		OrdType: okx.OrdLimit,
		Px:      "60000",
		Sz:      "0.001",
		ClOrdID: okx.NewClientOrderID(),
	})
	switch {
	case errors.Is(err, okx.ErrInsufficientBalance):
		fmt.Println("not enough funds")
	case err != nil:
		log.Fatal(err)
	default:
		fmt.Println("placed", res.OrdID)
	}
}

func ExampleAssetService_Convert() {
	ctx := context.Background()
	client := okx.NewClient(okx.WithCredentials("key", "secret", "passphrase"))

	// Spend 50 USDT on BTC from the funding account.
	pair, err := client.Asset.ConvertPair(ctx, okx.ConvertPairRequest{FromCcy: "USDT", ToCcy: "BTC"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("min", pair.QuoteCcyMin, "max", pair.QuoteCcyMax, pair.QuoteCcy)

	quote, err := client.Asset.ConvertQuote(ctx, okx.ConvertQuoteRequest{
		BaseCcy: "BTC", QuoteCcy: "USDT", Side: okx.SideBuy, RfqSz: "50", RfqSzCcy: "USDT",
	})
	if err != nil {
		log.Fatal(err)
	}
	trade, err := client.Asset.Convert(ctx, okx.ConvertRequest{
		QuoteID: quote.QuoteID, BaseCcy: "BTC", QuoteCcy: "USDT", Side: okx.SideBuy,
		Sz: quote.RfqSz, SzCcy: quote.RfqSzCcy,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(trade.State, trade.FillBaseSz, "BTC at", trade.FillPx)
}

func ExampleTradeService_AllFills() {
	client := okx.NewClient(okx.WithCredentials("key", "secret", "passphrase"))
	for fill, err := range client.Trade.AllFills(context.Background(), okx.FillsRequest{InstType: okx.InstSpot}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fill.InstID, fill.Side, fill.FillSz, fill.FillPx)
	}
}

func ExampleStream_Tickers() {
	stream := okx.NewStream()
	defer func() { _ = stream.Close() }()

	sub, err := stream.Tickers(context.Background(), "BTC-USDT")
	if err != nil {
		fmt.Println(err)
		return
	}
	for t := range sub.C {
		fmt.Println(t.Last)
	}
}

func ExampleBook() {
	ctx := context.Background()
	stream := okx.NewStream()
	defer func() { _ = stream.Close() }()

	sub, err := stream.OrderBook(ctx, "BTC-USDT", okx.Books)
	if err != nil {
		fmt.Println(err)
		return
	}
	var book okx.Book
	for update := range sub.C {
		if errors.Is(book.Apply(update), okx.ErrBookOutOfSync) {
			_ = sub.Resync(ctx)
			continue
		}
		if bid, ok := book.BestBid(); ok && book.Synced() {
			fmt.Println(bid.Px)
		}
	}
}
