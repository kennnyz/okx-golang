// Streams order and position updates and reconciles over REST after every
// reconnect, when pushes may have been missed.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	okx "github.com/kennnyz/okx-golang"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts := []okx.Option{
		okx.WithCredentials(os.Getenv("OKX_API_KEY"), os.Getenv("OKX_SECRET_KEY"), os.Getenv("OKX_PASSPHRASE")),
		okx.WithDemoTrading(),
		okx.WithLogger(slog.Default()),
	}
	client := okx.NewClient(opts...)
	reconcile := func() {
		orders, err := client.Trade.PendingOrders(ctx, okx.PendingOrdersRequest{})
		if err != nil {
			log.Println("reconcile:", err)
			return
		}
		fmt.Printf("reconciled: %d pending orders\n", len(orders))
	}

	stream := okx.NewStream(append(opts, okx.WithReconnectHook(func(string) { reconcile() }))...)
	defer func() { _ = stream.Close() }()

	orders, err := stream.Orders(ctx, okx.InstAny, "")
	if err != nil {
		return err
	}
	positions, err := stream.Positions(ctx, okx.InstAny, "")
	if err != nil {
		return err
	}
	reconcile()

	for {
		select {
		case <-ctx.Done():
			return nil
		case o := <-orders.C:
			fmt.Printf("order %s %s %s %s filled %s/%s @ %s\n", o.InstID, o.Side, o.OrdType, o.State, o.AccFillSz, o.Sz, o.AvgPx)
		case p := <-positions.C:
			fmt.Printf("position %s %s %s @ %s upl %s\n", p.InstID, p.PosSide, p.Pos, p.AvgPx, p.Upl)
		}
	}
}
