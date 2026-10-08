package okx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const Version = "0.1.0"

const maxResponseBytes = 16 << 20

// Client is the OKX v5 REST client. It is safe for concurrent use.
type Client struct {
	Market  *MarketService
	Public  *PublicService
	Account *AccountService
	Trade   *TradeService
	Asset   *AssetService

	cfg     config
	limiter *limiter
	skew    atomic.Int64
}

// NewClient creates a REST client. Without WithCredentials only public
// endpoints are available.
func NewClient(opts ...Option) *Client {
	c := &Client{cfg: newConfig(opts), limiter: newLimiter()}
	c.Market = &MarketService{c}
	c.Public = &PublicService{c}
	c.Account = &AccountService{c}
	c.Trade = &TradeService{c}
	c.Asset = &AssetService{c}
	return c
}

// SyncTime measures the offset between the local clock and OKX server time
// and applies it to request signatures. Requests rejected for an expired
// timestamp trigger it automatically.
func (c *Client) SyncTime(ctx context.Context) error {
	start := time.Now()
	server, err := c.Public.Time(ctx)
	if err != nil {
		return err
	}
	rtt := time.Since(start)
	c.skew.Store(int64(server.Sub(start.Add(rtt / 2))))
	return nil
}

func (c *Client) now() time.Time {
	return time.Now().Add(time.Duration(c.skew.Load()))
}

// Do calls any OKX v5 endpoint, including ones this package does not wrap.
// For GET, params is a struct with json tags or url.Values and is sent as the
// query string; for POST it is marshaled as the JSON body. out receives the
// "data" field of the response. Signing, rate limiting and retries apply.
func (c *Client) Do(ctx context.Context, method, path string, params, out any) error {
	return c.call(ctx, request{method: method, path: path, params: params, private: c.cfg.creds != nil}, out)
}

type request struct {
	method  string
	path    string
	params  any
	private bool
	batch   bool
	trade   bool
	limitBy string
}

func publicGet(path string, params any) request {
	return request{method: http.MethodGet, path: path, params: params}
}

func privateGet(path string, params any) request {
	return request{method: http.MethodGet, path: path, params: params, private: true}
}

func privatePost(path string, params any) request {
	return request{method: http.MethodPost, path: path, params: params, private: true}
}

func tradePost(path string, params any, instID string) request {
	return request{method: http.MethodPost, path: path, params: params, private: true, trade: true, limitBy: instID}
}

func list[T any](ctx context.Context, c *Client, r request) ([]T, error) {
	var out []T
	err := c.call(ctx, r, &out)
	return out, err
}

func one[T any](ctx context.Context, c *Client, r request) (T, error) {
	items, err := list[T](ctx, c, r)
	var zero T
	if err != nil {
		return zero, err
	}
	if len(items) == 0 {
		return zero, ErrEmptyResponse
	}
	return items[0], nil
}

func (c *Client) call(ctx context.Context, r request, out any) error {
	if r.private && c.cfg.creds == nil {
		return ErrNoCredentials
	}
	requestPath, body, err := encodeRequest(r)
	if err != nil {
		return err
	}
	synced := false
	for attempt := 0; ; attempt++ {
		if c.cfg.rateLimit {
			if err := c.limiter.wait(ctx, r.method+" "+r.path, r.limitBy); err != nil {
				return err
			}
		}
		err = c.send(ctx, r, requestPath, body, out)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrTimestampExpired) && !synced && r.private {
			synced = true
			if c.SyncTime(ctx) == nil {
				continue
			}
		}
		if attempt >= c.cfg.retries || !retryable(r.method, err) {
			return err
		}
		delay := backoff(attempt)
		c.cfg.logger.DebugContext(ctx, "okx: retrying request", "path", r.path, "attempt", attempt+1, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func encodeRequest(r request) (requestPath string, body []byte, err error) {
	requestPath = r.path
	if r.method == http.MethodGet {
		q, err := encodeQuery(r.params)
		if err != nil {
			return "", nil, err
		}
		if len(q) > 0 {
			requestPath += "?" + q.Encode()
		}
		return requestPath, nil, nil
	}
	if r.params == nil {
		return requestPath, []byte("{}"), nil
	}
	body, err = json.Marshal(r.params)
	if err != nil {
		return "", nil, fmt.Errorf("okx: encode body: %w", err)
	}
	return requestPath, body, nil
}

func retryable(method string, err error) bool {
	if errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTimestampExpired) {
		return true
	}
	if method != http.MethodGet {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return errors.Is(err, ErrServiceUnavailable) || apiErr.HTTPStatus >= 500
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func backoff(attempt int) time.Duration {
	d := 250 * time.Millisecond << attempt
	if d > 4*time.Second {
		d = 4 * time.Second
	}
	return d/2 + rand.N(d/2)
}

type envelope struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type itemStatus struct {
	SCode string `json:"sCode"`
	SMsg  string `json:"sMsg"`
}

func (c *Client) send(ctx context.Context, r request, requestPath string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, c.cfg.restURL+requestPath, reader)
	if err != nil {
		return fmt.Errorf("okx: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.cfg.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.demo {
		req.Header.Set("x-simulated-trading", "1")
	}
	if r.trade && c.cfg.requestTTL > 0 {
		req.Header.Set("expTime", strconv.FormatInt(time.Now().Add(c.cfg.requestTTL).UnixMilli(), 10))
	}
	if r.private {
		ts := c.now().UTC().Format("2006-01-02T15:04:05.000Z")
		creds := c.cfg.creds
		req.Header.Set("OK-ACCESS-KEY", creds.APIKey)
		req.Header.Set("OK-ACCESS-PASSPHRASE", creds.Passphrase)
		req.Header.Set("OK-ACCESS-TIMESTAMP", ts)
		req.Header.Set("OK-ACCESS-SIGN", sign(creds.SecretKey, ts+r.method+requestPath+string(body)))
	}

	resp, err := c.cfg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("okx: %s %s: %w", r.method, r.path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("okx: read response: %w", err)
	}
	return decodeResponse(resp.StatusCode, raw, r.batch, out)
}

func decodeResponse(status int, raw []byte, batch bool, out any) error {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Code == "" {
		if status != http.StatusOK {
			return &APIError{HTTPStatus: status, Code: strconv.Itoa(status), Msg: http.StatusText(status)}
		}
		return fmt.Errorf("okx: decode response: %w", err)
	}
	partial := batch && (env.Code == "1" || env.Code == "2") && hasData(env.Data)
	if env.Code != "0" && !partial {
		return &APIError{HTTPStatus: status, Code: env.Code, Msg: env.Msg, Items: itemErrors(env.Data)}
	}
	if out == nil || !hasData(env.Data) {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("okx: decode data: %w", err)
	}
	return nil
}

func hasData(d json.RawMessage) bool {
	return len(d) > 0 && string(d) != "null" && string(d) != "[]" && string(d) != "{}"
}

func itemErrors(data json.RawMessage) []ItemError {
	var items []itemStatus
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var errs []ItemError
	for _, it := range items {
		if it.SCode != "" && it.SCode != "0" {
			errs = append(errs, ItemError{Code: it.SCode, Msg: it.SMsg})
		}
	}
	return errs
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }
