package okx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	got := sign("secret", "2020-12-08T09:08:57.715ZGET/api/v5/account/balance?ccy=BTC")
	if want := "wpDvCwYCprcMQsQkxWJiWy+YADoQE4ep+OEKKLimMoY="; got != want {
		t.Fatalf("sign = %s, want %s", got, want)
	}
}

func newTestClient(t *testing.T, h http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(append([]Option{WithBaseURL(srv.URL), WithoutRateLimit()}, opts...)...)
}

func reply(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": "0", "msg": "", "data": data})
}

func TestPrivateRequestIsSigned(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ts := r.Header.Get("OK-ACCESS-TIMESTAMP")
		if _, err := time.Parse("2006-01-02T15:04:05.000Z", ts); err != nil {
			t.Errorf("timestamp %q: %v", ts, err)
		}
		want := sign("s3cret", ts+r.Method+r.URL.RequestURI()+string(body))
		if got := r.Header.Get("OK-ACCESS-SIGN"); got != want {
			t.Errorf("signature %s, want %s", got, want)
		}
		if r.Header.Get("OK-ACCESS-KEY") != "key" || r.Header.Get("OK-ACCESS-PASSPHRASE") != "pass" {
			t.Error("missing key or passphrase header")
		}
		if r.Header.Get("x-simulated-trading") != "1" {
			t.Error("missing demo trading header")
		}
		if r.Header.Get("expTime") == "" {
			t.Error("missing expTime header on trade request")
		}
		var req PlaceOrderRequest
		if err := json.Unmarshal(body, &req); err != nil || req.Sz != "0.01" || !req.ReduceOnly {
			t.Errorf("body %s: %v", body, err)
		}
		reply(w, []map[string]string{{"ordId": "42", "clOrdId": "abc", "sCode": "0", "ts": "1700000000000"}})
	}, WithCredentials("key", "s3cret", "pass"), WithDemoTrading(), WithRequestTTL(time.Second))

	res, err := c.Trade.PlaceOrder(context.Background(), PlaceOrderRequest{
		InstID: "BTC-USDT-SWAP", TdMode: TdCross, Side: SideSell, OrdType: OrdMarket, Sz: "0.01", ReduceOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrdID != "42" || res.Ts.UnixMilli() != 1700000000000 {
		t.Fatalf("result %+v", res)
	}
}

func TestPublicRequestQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OK-ACCESS-SIGN") != "" {
			t.Error("public request must not be signed")
		}
		want := url.Values{"instId": {"BTC-USDT"}, "bar": {"1H"}, "after": {"1700000000000"}, "limit": {"2"}}
		if got := r.URL.Query(); got.Encode() != want.Encode() {
			t.Errorf("query %v, want %v", got, want)
		}
		reply(w, [][]string{
			{"1700000000000", "1", "3", "0.5", "2", "10", "20", "20", "1"},
			{"1699996400000", "2", "3", "1", "1", "10", "20", "20", "0"},
		})
	})
	candles, err := c.Market.Candles(context.Background(), CandlesRequest{
		InstID: "BTC-USDT", Bar: Bar1H, After: TimeFromMillis(1700000000000), Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 || candles[0].High != "3" || !candles[0].Confirmed || candles[1].Confirmed {
		t.Fatalf("candles %+v", candles)
	}
}

func TestNoCredentials(t *testing.T) {
	c := NewClient()
	if _, err := c.Account.Balance(context.Background()); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err = %v", err)
	}
}

func TestOrderRejectionCarriesItemCode(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"1","msg":"All operations failed","data":[{"ordId":"","sCode":"51008","sMsg":"Insufficient balance"}]}`))
	}, WithCredentials("k", "s", "p"))

	_, err := c.Trade.PlaceOrder(context.Background(), PlaceOrderRequest{InstID: "BTC-USDT"})
	if !errors.Is(err, ErrInsufficientBalance) || !HasCode(err, "51008") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "51008: Insufficient balance") {
		t.Fatalf("message %q", err.Error())
	}
}

func TestBatchPartialSuccess(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"2","msg":"","data":[{"ordId":"1","sCode":"0"},{"ordId":"","sCode":"51121","sMsg":"lot size"}]}`))
	}, WithCredentials("k", "s", "p"))

	res, err := c.Trade.PlaceOrders(context.Background(), []PlaceOrderRequest{{InstID: "A"}, {InstID: "A"}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err() != nil || !HasCode(res[1].Err(), "51121") {
		t.Fatalf("results %+v", res)
	}
}

func TestRetriesRateLimit(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":"50011","msg":"Too Many Requests","data":[]}`))
			return
		}
		reply(w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}, WithCredentials("k", "s", "p"))

	if _, err := c.Trade.PlaceOrder(context.Background(), PlaceOrderRequest{InstID: "X"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestDoesNotRetryUnsafePost(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}, WithCredentials("k", "s", "p"))

	_, err := c.Trade.PlaceOrder(context.Background(), PlaceOrderRequest{InstID: "X"})
	if !errors.Is(err, ErrServiceUnavailable) || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
}

func TestRetriesGetOnServerError(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		reply(w, []map[string]string{{"instId": "BTC-USDT", "last": "1"}})
	})
	if _, err := c.Market.Ticker(context.Background(), "BTC-USDT"); err != nil || calls.Load() != 2 {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
}

func TestExpiredTimestampResyncsClock(t *testing.T) {
	serverTime := time.Now().Add(time.Hour)
	var private atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v5/public/time" {
			reply(w, []map[string]string{{"ts": NumberFromInt(serverTime.UnixMilli()).String()}})
			return
		}
		private.Add(1)
		ts, _ := time.Parse("2006-01-02T15:04:05.000Z", r.Header.Get("OK-ACCESS-TIMESTAMP"))
		if ts.Sub(serverTime).Abs() > time.Minute {
			_, _ = w.Write([]byte(`{"code":"50102","msg":"Timestamp request expired","data":[]}`))
			return
		}
		reply(w, []map[string]string{{"totalEq": "10"}})
	}, WithCredentials("k", "s", "p"), WithRetry(0))

	bal, err := c.Account.Balance(context.Background())
	if err != nil || bal.TotalEq != "10" || private.Load() != 2 {
		t.Fatalf("balance %+v, err %v, calls %d", bal, err, private.Load())
	}
}

func TestHTTPErrorWithoutEnvelope(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}, WithRetry(0))
	_, err := c.Market.Ticker(context.Background(), "X")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 401 || !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v", err)
	}
}

func TestDoRawEndpoint(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/copytrading/current-subpositions" || r.URL.Query().Get("instType") != "SWAP" {
			t.Errorf("unexpected request %s", r.URL)
		}
		reply(w, []map[string]string{{"subPosId": "7"}})
	}, WithCredentials("k", "s", "p"))

	var out []struct {
		SubPosID string `json:"subPosId"`
	}
	err := c.Do(context.Background(), http.MethodGet, "/api/v5/copytrading/current-subpositions", url.Values{"instType": {"SWAP"}}, &out)
	if err != nil || len(out) != 1 || out[0].SubPosID != "7" {
		t.Fatalf("out %+v, err %v", out, err)
	}
}

func TestAllFillsPaginates(t *testing.T) {
	pages := map[string][]map[string]string{
		"":   {{"billId": "30"}, {"billId": "20"}},
		"20": {{"billId": "10"}},
		"10": {},
	}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("instType") != "SPOT" {
			t.Error("filter not kept across pages")
		}
		reply(w, pages[r.URL.Query().Get("after")])
	}, WithCredentials("k", "s", "p"))

	var ids []string
	for f, err := range c.Trade.AllFills(context.Background(), FillsRequest{InstType: InstSpot}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, f.BillID)
	}
	if strings.Join(ids, ",") != "30,20,10" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestEncodeQuery(t *testing.T) {
	q, err := encodeQuery(struct {
		A string   `json:"a,omitempty"`
		B int      `json:"b,omitempty"`
		C bool     `json:"c,omitempty"`
		D Time     `json:"d,omitempty"`
		E []string `json:"e,omitempty"`
		F Number   `json:"f"`
		G string   `json:"-"`
	}{A: "x", C: true, D: TimeFromMillis(5), E: []string{"1", "2"}, G: "skip"})
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Encode(); got != "a=x&c=true&d=5&e=1%2C2" {
		t.Fatalf("query %s", got)
	}
}

func TestLimiterWaits(t *testing.T) {
	l := newLimiter()
	limits["GET /test"] = rule{2, 200 * time.Millisecond}
	defer delete(limits, "GET /test")

	start := time.Now()
	for range 4 {
		if err := l.wait(context.Background(), "GET /test", ""); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Fatalf("4 requests at 2/200ms took %v", d)
	}
	if err := l.wait(context.Background(), "GET /unlimited", ""); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range 3 {
		if err := l.wait(ctx, "GET /test", "other"); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("cancelled context did not stop waiting")
}

func TestTradeRequestNotRetriedAfterExpiry(t *testing.T) {
	var calls atomic.Int32
	var expTimes []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		expTimes = append(expTimes, r.Header.Get("expTime"))
		_, _ = w.Write([]byte(`{"code":"50011","msg":"Too Many Requests","data":[]}`))
	}, WithCredentials("k", "s", "p"), WithRequestTTL(50*time.Millisecond), WithRetry(5))

	_, err := c.Trade.PlaceOrder(context.Background(), PlaceOrderRequest{InstID: "X"})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() > 2 {
		t.Fatalf("retried %d times past the request TTL", calls.Load())
	}
	for _, e := range expTimes {
		if e != expTimes[0] {
			t.Fatalf("expTime changed between attempts: %v", expTimes)
		}
	}
}

func TestBatchCostsOneTokenPerOrder(t *testing.T) {
	l := newLimiter()
	limits["POST /batch"] = rule{4, time.Second}
	defer delete(limits, "POST /batch")

	start := time.Now()
	for range 2 {
		if err := l.waitCost(context.Background(), "POST /batch", map[string]int{"A": 3}); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 400*time.Millisecond {
		t.Fatalf("6 orders at 4/s went through in %v", d)
	}
	if err := l.waitCost(context.Background(), "POST /batch", map[string]int{"B": 4}); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 900*time.Millisecond {
		t.Fatalf("other instrument waited for A's bucket: %v", d)
	}
}
