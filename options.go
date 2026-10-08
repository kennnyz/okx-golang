package okx

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Region selects the OKX entity whose domains the client talks to. Accounts
// registered with OKX EEA or OKX US must use their own region.
type Region int

const (
	RegionGlobal Region = iota
	RegionEEA
	RegionUS
)

type hosts struct{ rest, ws, wsDemo string }

var regionHosts = map[Region]hosts{
	RegionGlobal: {"https://openapi.okx.com", "wss://ws.okx.com", "wss://wspap.okx.com"},
	RegionEEA:    {"https://eea.okx.com", "wss://wseea.okx.com", "wss://wseeapap.okx.com"},
	RegionUS:     {"https://us.okx.com", "wss://wsus.okx.com", "wss://wsuspap.okx.com"},
}

// Credentials is an OKX API key. Create one at okx.com → Profile → API keys;
// demo trading keys are created separately inside demo trading mode.
type Credentials struct {
	APIKey     string
	SecretKey  string
	Passphrase string
}

type config struct {
	creds      *Credentials
	region     Region
	demo       bool
	restURL    string
	wsURL      string
	httpClient *http.Client
	retries    int
	rateLimit  bool
	requestTTL time.Duration
	logger     *slog.Logger
	userAgent  string

	streamBuffer int
	onReconnect  func(url string)
}

func newConfig(opts []Option) config {
	c := config{
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		retries:      3,
		rateLimit:    true,
		logger:       slog.New(discardHandler{}),
		userAgent:    "okx-golang/" + Version,
		streamBuffer: 256,
	}
	for _, opt := range opts {
		opt(&c)
	}
	h := regionHosts[c.region]
	if c.restURL == "" {
		c.restURL = h.rest
	}
	c.restURL = strings.TrimRight(c.restURL, "/")
	if c.wsURL == "" {
		c.wsURL = h.ws
		if c.demo {
			c.wsURL = h.wsDemo
		}
	}
	c.wsURL = strings.TrimRight(c.wsURL, "/")
	return c
}

// Option configures a Client or a Stream.
type Option func(*config)

// WithCredentials enables private endpoints and channels.
func WithCredentials(apiKey, secretKey, passphrase string) Option {
	return func(c *config) {
		c.creds = &Credentials{APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase}
	}
}

// WithDemoTrading routes every request to OKX demo trading. Demo trading needs
// a key created in demo mode; production keys are rejected.
func WithDemoTrading() Option { return func(c *config) { c.demo = true } }

// WithRegion selects the OKX domains for EEA or US accounts.
func WithRegion(r Region) Option { return func(c *config) { c.region = r } }

// WithBaseURL overrides the REST base URL, e.g. "https://www.okx.com".
func WithBaseURL(url string) Option { return func(c *config) { c.restURL = url } }

// WithWebSocketURL overrides the WebSocket host, e.g. "wss://ws.okx.com:8443".
// Paths /ws/v5/public, /ws/v5/private and /ws/v5/business are appended.
func WithWebSocketURL(url string) Option { return func(c *config) { c.wsURL = url } }

// WithHTTPClient replaces the default HTTP client (15s timeout).
func WithHTTPClient(hc *http.Client) Option { return func(c *config) { c.httpClient = hc } }

// WithRetry sets how many times a failed request is retried. Rate-limit
// rejections and expired timestamps are retried for any method because OKX
// did not process them; network errors and 5xx are retried only for GET.
// Zero disables retries. Default is 3.
func WithRetry(max int) Option { return func(c *config) { c.retries = max } }

// WithoutRateLimit disables the client-side per-endpoint rate limiter.
func WithoutRateLimit() Option { return func(c *config) { c.rateLimit = false } }

// WithRequestTTL makes OKX drop trade requests (place, amend, cancel) that
// reach the matching engine later than ttl after being sent. Useful for
// strategies where a late order is worse than no order.
func WithRequestTTL(ttl time.Duration) Option { return func(c *config) { c.requestTTL = ttl } }

// WithLogger receives retry and reconnect events at debug and warn levels.
func WithLogger(l *slog.Logger) Option { return func(c *config) { c.logger = l } }

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option { return func(c *config) { c.userAgent = ua } }

// WithStreamBuffer sets the channel capacity of each Stream subscription.
// Default is 256.
func WithStreamBuffer(n int) Option { return func(c *config) { c.streamBuffer = n } }

// WithReconnectHook is called by Stream after a connection is re-established
// and all subscriptions are restored. Messages may have been missed while
// disconnected, so this is the place to reconcile state over REST.
func WithReconnectHook(fn func(url string)) Option { return func(c *config) { c.onReconnect = fn } }
