package okx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestConvertCurrencies(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		reply(w, []map[string]string{{"ccy": "BTC", "min": "", "max": ""}, {"ccy": "ETH", "min": "", "max": ""}})
	}, WithCredentials("k", "s", "p"))

	got, err := c.Asset.ConvertCurrencies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"BTC", "ETH"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// OKX allows one quote per pair every 5 seconds, so a second quote for the
// same pair waits while another pair goes through at once.
func TestConvertQuoteLimitedPerPair(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(w, []map[string]string{{"quoteId": "q"}})
	}))
	t.Cleanup(srv.Close)
	c := NewClient(WithBaseURL(srv.URL), WithCredentials("k", "s", "p"))

	quote := func(base string, timeout time.Duration) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err := c.Asset.ConvertQuote(ctx, ConvertQuoteRequest{BaseCcy: base, QuoteCcy: "USDT", Side: SideBuy, RfqSz: "10", RfqSzCcy: "USDT"})
		return err
	}

	if err := quote("BTC", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := quote("ETH", time.Second); err != nil {
		t.Fatalf("other pair: %v", err)
	}
	if err := quote("BTC", 200*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("same pair: got %v, want context.DeadlineExceeded", err)
	}
}
