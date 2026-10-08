package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	stream := okx.NewStream()
	defer func() { _ = stream.Close() }()

	sub, err := stream.OrderBook(ctx, "BTC-USDT", okx.Books)
	if err != nil {
		return err
	}

	var book okx.Book
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-sub.C:
			if err := book.Apply(update); errors.Is(err, okx.ErrBookOutOfSync) {
				log.Println("sequence gap, requesting a fresh snapshot")
				if err := sub.Resync(ctx); err != nil {
					return err
				}
				continue
			}
			if !book.Synced() {
				continue
			}
			bid, _ := book.BestBid()
			ask, _ := book.BestAsk()
			fmt.Printf("\rbid %s × %s | ask %s × %s | levels %d/%d   ",
				bid.Px, bid.Sz, ask.Px, ask.Sz, len(book.Bids), len(book.Asks))
		}
	}
}
