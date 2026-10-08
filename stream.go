package okx

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrStreamClosed = errors.New("okx: stream closed")
	ErrDisconnected = errors.New("okx: websocket disconnected")
)

// Stream is a WebSocket client for OKX push channels and WebSocket trading.
// It opens connections lazily, keeps them alive with pings, and on
// disconnect reconnects, logs in again and restores every subscription. It
// is safe for concurrent use.
type Stream struct {
	cfg     config
	ctx     context.Context
	cancel  context.CancelFunc
	ids     atomic.Uint64
	rest    *Client
	limiter *limiter
	wg      sync.WaitGroup

	mu     sync.Mutex
	conns  map[endpoint]*wsConn
	closed bool
}

// NewStream creates a Stream. No connection is made until the first
// subscription or WebSocket trading call.
func NewStream(opts ...Option) *Stream {
	cfg := newConfig(opts)
	ctx, cancel := context.WithCancel(context.Background())
	return &Stream{
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		rest:    newClient(cfg),
		limiter: newLimiter(),
		conns:   map[endpoint]*wsConn{},
	}
}

// Close disconnects, closes the channels of all subscriptions and waits for
// background goroutines to exit.
func (s *Stream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	conns := s.conns
	s.mu.Unlock()

	s.cancel()
	for _, c := range conns {
		c.shutdown()
	}
	s.wg.Wait()
	return nil
}

// SyncTime aligns login and request expiry timestamps with OKX server time.
// It runs automatically when OKX rejects a login timestamp.
func (s *Stream) SyncTime(ctx context.Context) error { return s.rest.SyncTime(ctx) }

func (s *Stream) now() time.Time { return s.rest.now() }

func (s *Stream) nextID() string { return strconv.FormatUint(s.ids.Add(1), 10) }

func (s *Stream) goroutine(fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func (s *Stream) conn(ep endpoint) (*wsConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrStreamClosed
	}
	c := s.conns[ep]
	if c == nil {
		c = newWSConn(s, ep)
		s.conns[ep] = c
		s.goroutine(c.run)
	}
	return c, nil
}

// Arg identifies a channel subscription. Only the fields the channel
// accepts should be set.
type Arg struct {
	Channel     string   `json:"channel"`
	InstType    InstType `json:"instType,omitempty"`
	InstFamily  string   `json:"instFamily,omitempty"`
	InstID      string   `json:"instId,omitempty"`
	Ccy         string   `json:"ccy,omitempty"`
	AlgoID      string   `json:"algoId,omitempty"`
	ExtraParams string   `json:"extraParams,omitempty"`
}

func (a Arg) key() string {
	return strings.Join([]string{a.Channel, string(a.InstType), a.InstFamily, a.InstID, a.Ccy, a.AlgoID}, "|")
}

// Push is one raw message of a channel. Action is "snapshot" or "update" for
// incremental channels such as order books.
type Push struct {
	Arg    Arg             `json:"arg"`
	Action string          `json:"action,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// Subscription delivers the messages of one channel on C in order. A slow
// reader never blocks the connection or other subscriptions: undelivered
// messages queue in memory until read. C is closed after Unsubscribe, after
// Stream.Close, or when OKX rejects the subscription on reconnect.
type Subscription[T any] struct {
	C <-chan T

	ch     chan T
	mu     sync.Mutex
	queue  []T
	wake   chan struct{}
	done   chan struct{}
	once   sync.Once
	decode func(Push) ([]T, error)
	conn   *wsConn
	arg    Arg
}

// Arg returns the channel this subscription is attached to.
func (s *Subscription[T]) Arg() Arg { return s.arg }

// Unsubscribe stops delivery, closes C and unsubscribes from OKX when no
// other subscription shares the channel.
func (s *Subscription[T]) Unsubscribe(ctx context.Context) error {
	s.shutdown()
	return s.conn.remove(ctx, s.arg, s)
}

// Resync unsubscribes and subscribes again on the server so incremental
// channels start over with a fresh snapshot. Use it after Book.Apply
// reports ErrBookOutOfSync.
func (s *Subscription[T]) Resync(ctx context.Context) error {
	return s.conn.resync(ctx, s.arg)
}

func (s *Subscription[T]) deliver(p Push) {
	items, err := s.decode(p)
	if err != nil {
		s.conn.s.cfg.logger.Warn("okx: decode push", "channel", p.Arg.Channel, "err", err)
		return
	}
	select {
	case <-s.done:
		return
	default:
	}
	s.mu.Lock()
	s.queue = append(s.queue, items...)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Subscription[T]) pump() {
	defer close(s.ch)
	for {
		s.mu.Lock()
		items := s.queue
		s.queue = nil
		s.mu.Unlock()
		for _, it := range items {
			select {
			case s.ch <- it:
			case <-s.done:
				return
			}
		}
		select {
		case <-s.wake:
		case <-s.done:
			return
		}
	}
}

func (s *Subscription[T]) shutdown() {
	s.once.Do(func() { close(s.done) })
}

type sink interface {
	deliver(Push)
	shutdown()
}

func subscribe[T any](ctx context.Context, s *Stream, arg Arg, decode func(Push) ([]T, error)) (*Subscription[T], error) {
	ep := channelEndpoint(arg.Channel)
	if ep.private() && s.cfg.creds == nil {
		return nil, ErrNoCredentials
	}
	c, err := s.conn(ep)
	if err != nil {
		return nil, err
	}
	ch := make(chan T, s.cfg.streamBuffer)
	sub := &Subscription[T]{
		C: ch, ch: ch,
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
		decode: decode,
		conn:   c,
		arg:    arg,
	}
	s.goroutine(sub.pump)
	if err := c.add(ctx, arg, sub); err != nil {
		sub.shutdown()
		return nil, err
	}
	return sub, nil
}

func decodeData[T any](p Push) ([]T, error) {
	var out []T
	err := json.Unmarshal(p.Data, &out)
	return out, err
}

// Subscribe subscribes to any channel and delivers raw pushes. Prefer the
// typed methods such as Tickers or Orders when available.
func (s *Stream) Subscribe(ctx context.Context, arg Arg) (*Subscription[Push], error) {
	return subscribe(ctx, s, arg, func(p Push) ([]Push, error) { return []Push{p}, nil })
}

type endpoint int

const (
	endpointPublic endpoint = iota
	endpointPrivate
	endpointBusiness
	endpointBusinessPrivate
)

func (e endpoint) path() string {
	switch e {
	case endpointPrivate:
		return "/ws/v5/private"
	case endpointBusiness, endpointBusinessPrivate:
		return "/ws/v5/business"
	}
	return "/ws/v5/public"
}

func (e endpoint) private() bool { return e == endpointPrivate || e == endpointBusinessPrivate }

var channelEndpoints = map[string]endpoint{
	"account":                   endpointPrivate,
	"positions":                 endpointPrivate,
	"balance_and_position":      endpointPrivate,
	"orders":                    endpointPrivate,
	"fills":                     endpointPrivate,
	"liquidation-warning":       endpointPrivate,
	"account-greeks":            endpointPrivate,
	"trades-all":                endpointBusiness,
	"sprd-public-trades":        endpointBusiness,
	"sprd-bbo-tbt":              endpointBusiness,
	"sprd-books5":               endpointBusiness,
	"sprd-books-l2-tbt":         endpointBusiness,
	"sprd-tickers":              endpointBusiness,
	"public-struc-block-trades": endpointBusiness,
	"public-block-trades":       endpointBusiness,
	"block-tickers":             endpointBusiness,
	"orders-algo":               endpointBusinessPrivate,
	"algo-advance":              endpointBusinessPrivate,
	"deposit-info":              endpointBusinessPrivate,
	"withdrawal-info":           endpointBusinessPrivate,
	"sprd-orders":               endpointBusinessPrivate,
	"sprd-trades":               endpointBusinessPrivate,
	"rfqs":                      endpointBusinessPrivate,
	"quotes":                    endpointBusinessPrivate,
	"struc-block-trades":        endpointBusinessPrivate,
	"economic-calendar":         endpointBusinessPrivate,
}

func channelEndpoint(channel string) endpoint {
	if ep, ok := channelEndpoints[channel]; ok {
		return ep
	}
	switch {
	case strings.HasPrefix(channel, "candle"),
		strings.HasPrefix(channel, "mark-price-candle"),
		strings.HasPrefix(channel, "index-candle"),
		strings.HasPrefix(channel, "sprd-candle"):
		return endpointBusiness
	case strings.HasPrefix(channel, "grid-"):
		return endpointBusinessPrivate
	}
	return endpointPublic
}
