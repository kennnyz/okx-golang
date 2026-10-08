package main

import (
	"context"
	"fmt"
	"log"
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

	client := okx.NewClient()

	ticker, err := client.Market.Ticker(ctx, "BTC-USDT")
	if err != nil {
		return err
	}
	fmt.Printf("BTC-USDT last %s, 24h high %s, low %s\n", ticker.Last, ticker.High24h, ticker.Low24h)

	funding, err := client.Public.FundingRate(ctx, "BTC-USDT-SWAP")
	if err != nil {
		return err
	}
	fmt.Printf("BTC-USDT-SWAP funding %s, next at %s\n", funding.FundingRate, funding.FundingTime.Format(time.RFC3339))

	since := time.Now().AddDate(0, 0, -30)
	var closes []float64
	for candle, err := range client.Market.AllHistoryCandles(ctx, okx.CandlesRequest{InstID: "BTC-USDT", Bar: okx.Bar4H}) {
		if err != nil {
			return err
		}
		if candle.Ts.Before(since) {
			break
		}
		closes = append(closes, candle.Close.Float64())
	}
	fmt.Printf("loaded %d 4H candles for the last 30 days\n", len(closes))
	return nil
}
