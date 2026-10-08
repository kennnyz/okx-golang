// Places a limit buy with an attached take-profit and stop-loss on OKX demo
// trading, then cancels it. Create a demo API key in OKX → Demo trading →
// API and export OKX_API_KEY, OKX_SECRET_KEY and OKX_PASSPHRASE.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	okx "github.com/kennnyz/okx-golang"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := okx.NewClient(
		okx.WithCredentials(os.Getenv("OKX_API_KEY"), os.Getenv("OKX_SECRET_KEY"), os.Getenv("OKX_PASSPHRASE")),
		okx.WithDemoTrading(),
	)

	if _, err := client.Trade.CancelAllAfter(ctx, 60*time.Second); err != nil {
		return err
	}

	ticker, err := client.Market.Ticker(ctx, "BTC-USDT")
	if err != nil {
		return err
	}
	entry := math.Round(ticker.Last.Float64() * 0.95)

	order, err := client.Trade.PlaceOrder(ctx, okx.PlaceOrderRequest{
		InstID:  "BTC-USDT",
		TdMode:  okx.TdCash,
		Side:    okx.SideBuy,
		OrdType: okx.OrdLimit,
		Px:      okx.NumberFromFloat(entry),
		Sz:      "0.001",
		ClOrdID: okx.NewClientOrderID(),
		AttachAlgoOrds: []okx.AttachAlgoOrder{{
			TpTriggerPx: okx.NumberFromFloat(math.Round(entry * 1.05)),
			TpOrdPx:     "-1",
			SlTriggerPx: okx.NumberFromFloat(math.Round(entry * 0.97)),
			SlOrdPx:     "-1",
		}},
	})
	if err != nil {
		return err
	}
	fmt.Println("placed", order.OrdID)

	placed, err := client.Trade.Order(ctx, okx.OrderRequest{InstID: "BTC-USDT", OrdID: order.OrdID})
	if err != nil {
		return err
	}
	fmt.Printf("state %s, price %s, size %s\n", placed.State, placed.Px, placed.Sz)

	if _, err := client.Trade.CancelOrder(ctx, okx.CancelOrderRequest{InstID: "BTC-USDT", OrdID: order.OrdID}); err != nil {
		return err
	}
	fmt.Println("cancelled")
	return nil
}
