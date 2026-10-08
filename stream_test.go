package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type fakeOKX struct {
	t      *testing.T
	srv    *httptest.Server
	ops    chan fakeOp
	mu     sync.Mutex
	conns  []*websocket.Conn
	reject map[string]bool
}

type fakeOp struct {
	path string
	op   string
	args []json.RawMessage
}

func newFakeOKX(t *testing.T) *fakeOKX {
	f := &fakeOKX{t: t, ops: make(chan fakeOp, 100)}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeOKX) url() string { return "ws" + strings.TrimPrefix(f.srv.URL, "http") }

func (f *fakeOKX) serve(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	f.mu.Lock()
	f.conns = append(f.conns, ws)
	f.mu.Unlock()
	ctx := context.Background()
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return
		}
		if string(data) == "ping" {
			_ = ws.Write(ctx, websocket.MessageText, []byte("pong"))
			continue
		}
		var m struct {
			ID   string            `json:"id"`
			Op   string            `json:"op"`
			Args []json.RawMessage `json:"args"`
		}
		_ = json.Unmarshal(data, &m)
		f.ops <- fakeOp{path: r.URL.Path, op: m.Op, args: m.Args}
		f.respond(ws, m.ID, m.Op, m.Args)
	}
}

func (f *fakeOKX) respond(ws *websocket.Conn, id, op string, args []json.RawMessage) {
	send := func(v any) {
		b, _ := json.Marshal(v)
		_ = ws.Write(context.Background(), websocket.MessageText, b)
	}
	switch op {
	case "login":
		var a map[string]string
		_ = json.Unmarshal(args[0], &a)
		if a["sign"] != sign("secret", a["timestamp"]+"GET/users/self/verify") {
			send(map[string]string{"event": "error", "code": "60009", "msg": "Login failed."})
			return
		}
		send(map[string]string{"event": "login", "code": "0"})
	case "subscribe", "unsubscribe":
		var a Arg
		_ = json.Unmarshal(args[0], &a)
		f.mu.Lock()
		rejected := f.reject[a.InstID]
		f.mu.Unlock()
		if a.Channel == "bad" || (op == "subscribe" && rejected) {
			send(map[string]string{"id": id, "event": "error", "code": "60018", "msg": "channel doesn't exist"})
			return
		}
		send(map[string]any{"id": id, "event": op, "arg": a})
	case "order":
		var a PlaceOrderRequest
		_ = json.Unmarshal(args[0], &a)
		if a.InstID == "FAIL" {
			send(map[string]any{"id": id, "op": op, "code": "1", "msg": "", "data": []map[string]string{{"sCode": "51008", "sMsg": "Insufficient balance"}}})
			return
		}
		send(map[string]any{"id": id, "op": op, "code": "0", "msg": "", "data": []map[string]string{{"ordId": "777", "clOrdId": a.ClOrdID, "sCode": "0"}}})
	}
}

func (f *fakeOKX) push(t *testing.T, conn int, msg string) {
	t.Helper()
	f.mu.Lock()
	ws := f.conns[conn]
	f.mu.Unlock()
	if err := ws.Write(context.Background(), websocket.MessageText, []byte(msg)); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeOKX) rejectInst(instID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reject == nil {
		f.reject = map[string]bool{}
	}
	f.reject[instID] = true
}

func (f *fakeOKX) dropAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ws := range f.conns {
		_ = ws.CloseNow()
	}
}

func (f *fakeOKX) expect(t *testing.T, path, op string) fakeOp {
	t.Helper()
	select {
	case o := <-f.ops:
		if o.path != path || o.op != op {
			t.Fatalf("got %s %s, want %s %s", o.path, o.op, path, op)
		}
		return o
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s %s", path, op)
	}
	return fakeOp{}
}

func (f *fakeOKX) expectNone(t *testing.T) {
	t.Helper()
	select {
	case o := <-f.ops:
		t.Fatalf("unexpected %s %s", o.path, o.op)
	case <-time.After(100 * time.Millisecond):
	}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for push")
	}
	var zero T
	return zero
}

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

const tickerPush = `{"arg":{"channel":"tickers","instId":"BTC-USDT"},"data":[{"instId":"BTC-USDT","last":"%s","ts":"1700000000000"}]}`

func TestStreamDeliversTypedPushes(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	a, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, "/ws/v5/public", "subscribe")
	b, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	f.expectNone(t)

	f.push(t, 0, strings.Replace(tickerPush, "%s", "65000.1", 1))
	for _, sub := range []*Subscription[Ticker]{a, b} {
		if tk := receive(t, sub.C); tk.Last != "65000.1" || tk.Ts.UnixMilli() != 1700000000000 {
			t.Fatalf("ticker %+v", tk)
		}
	}

	if err := a.Unsubscribe(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-a.C; ok {
		t.Fatal("C not closed after Unsubscribe")
	}
	f.expectNone(t)
	if err := b.Unsubscribe(ctx); err != nil {
		t.Fatal(err)
	}
	f.expect(t, "/ws/v5/public", "unsubscribe")
}

func TestStreamRoutesChannelsToEndpoints(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	candles, err := s.Candles(ctx, "BTC-USDT", Bar1m)
	if err != nil {
		t.Fatal(err)
	}
	o := f.expect(t, "/ws/v5/business", "subscribe")
	if !strings.Contains(string(o.args[0]), `"candle1m"`) {
		t.Fatalf("args %s", o.args[0])
	}
	f.push(t, 0, `{"arg":{"channel":"candle1m","instId":"BTC-USDT"},"data":[["1700000000000","1","2","0.5","1.5","10","20","30","0"]]}`)
	if c := receive(t, candles.C); c.Close != "1.5" || c.Confirmed {
		t.Fatalf("candle %+v", c)
	}

	if _, err := s.Orders(ctx, InstAny, ""); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("private channel without credentials: %v", err)
	}
}

func TestStreamSubscribeError(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()))
	defer func() { _ = s.Close() }()

	_, err := s.Subscribe(testCtx(t), Arg{Channel: "bad"})
	if !HasCode(err, "60018") {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamReconnectsAndResubscribes(t *testing.T) {
	f := newFakeOKX(t)
	reconnected := make(chan string, 1)
	s := NewStream(WithWebSocketURL(f.url()), WithReconnectHook(func(url string) { reconnected <- url }))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	sub, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, "/ws/v5/public", "subscribe")

	f.dropAll()
	o := f.expect(t, "/ws/v5/public", "subscribe")
	if !strings.Contains(string(o.args[0]), `"BTC-USDT"`) {
		t.Fatalf("resubscribed %s", o.args[0])
	}
	if url := receive(t, reconnected); !strings.HasSuffix(url, "/ws/v5/public") {
		t.Fatalf("hook url %s", url)
	}
	f.push(t, 1, strings.Replace(tickerPush, "%s", "1", 1))
	if tk := receive(t, sub.C); tk.Last != "1" {
		t.Fatalf("ticker after reconnect %+v", tk)
	}
}

func TestStreamTradingOverPrivateConnection(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()), WithCredentials("key", "secret", "pass"))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	res, err := s.PlaceOrder(ctx, PlaceOrderRequest{InstID: "BTC-USDT", ClOrdID: "c1", Sz: "1"})
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, "/ws/v5/private", "login")
	f.expect(t, "/ws/v5/private", "order")
	if res.OrdID != "777" || res.ClOrdID != "c1" {
		t.Fatalf("result %+v", res)
	}

	_, err = s.PlaceOrder(ctx, PlaceOrderRequest{InstID: "FAIL"})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamLoginFailure(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()), WithCredentials("key", "wrong", "pass"))
	defer func() { _ = s.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := s.Orders(ctx, InstAny, "")
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamCloseClosesSubscriptions(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()))
	sub, err := s.Tickers(testCtx(t), "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	select {
	case _, ok := <-sub.C:
		if ok {
			t.Fatal("unexpected message")
		}
	case <-time.After(time.Second):
		t.Fatal("C not closed")
	}
	if _, err := s.Tickers(context.Background(), "ETH-USDT"); !errors.Is(err, ErrStreamClosed) {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamSlowSubscriberDoesNotBlockConnection(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()), WithStreamBuffer(1))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	slow, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, "/ws/v5/public", "subscribe")
	for _, px := range []string{"1", "2", "3", "4", "5"} {
		f.push(t, 0, strings.Replace(tickerPush, "%s", px, 1))
	}

	quick, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := s.Tickers(quick, "ETH-USDT"); err != nil {
		t.Fatalf("subscribe while another subscriber is not reading: %v", err)
	}
	for _, want := range []string{"1", "2", "3", "4", "5"} {
		if tk := receive(t, slow.C); string(tk.Last) != want {
			t.Fatalf("got %s, want %s", tk.Last, want)
		}
	}
}

func TestStreamDropsSubscriptionRejectedOnReconnect(t *testing.T) {
	f := newFakeOKX(t)
	reconnected := make(chan string, 1)
	s := NewStream(WithWebSocketURL(f.url()), WithReconnectHook(func(url string) { reconnected <- url }))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	gone, err := s.Tickers(ctx, "GONE-USDT")
	if err != nil {
		t.Fatal(err)
	}
	kept, err := s.Tickers(ctx, "BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	f.rejectInst("GONE-USDT")
	f.dropAll()
	receive(t, reconnected)

	select {
	case _, ok := <-gone.C:
		if ok {
			t.Fatal("unexpected message")
		}
	case <-time.After(time.Second):
		t.Fatal("rejected subscription was not closed")
	}
	f.push(t, 1, strings.Replace(tickerPush, "%s", "7", 1))
	if tk := receive(t, kept.C); tk.Last != "7" {
		t.Fatalf("ticker %+v", tk)
	}
}

func TestStreamConcurrentSubscribersShareChannel(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()))
	defer func() { _ = s.Close() }()
	ctx := testCtx(t)

	subs := make([]*Subscription[Ticker], 8)
	var wg sync.WaitGroup
	for i := range subs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub, err := s.Tickers(ctx, "BTC-USDT")
			if err != nil {
				t.Error(err)
			}
			subs[i] = sub
		}()
	}
	wg.Wait()
	f.expect(t, "/ws/v5/public", "subscribe")
	f.expectNone(t)
	f.push(t, 0, strings.Replace(tickerPush, "%s", "9", 1))
	for _, sub := range subs {
		if tk := receive(t, sub.C); tk.Last != "9" {
			t.Fatalf("ticker %+v", tk)
		}
	}
}

func TestStreamPublicBusinessChannelsSkipLogin(t *testing.T) {
	f := newFakeOKX(t)
	s := NewStream(WithWebSocketURL(f.url()), WithCredentials("key", "wrong", "pass"))
	defer func() { _ = s.Close() }()

	if _, err := s.Candles(testCtx(t), "BTC-USDT", Bar1H); err != nil {
		t.Fatalf("public candles with bad credentials: %v", err)
	}
	f.expect(t, "/ws/v5/business", "subscribe")
}
