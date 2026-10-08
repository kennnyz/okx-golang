package okx

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrStreamClosed = errors.New("okx: stream closed")
	ErrDisconnected = errors.New("okx: websocket disconnected")
)

// Stream is a WebSocket client for OKX push channels and WebSocket trading.
// It opens the public, private and business connections lazily, keeps them
// alive with pings, and on disconnect reconnects, logs in again and restores
// every subscription. It is safe for concurrent use.
type Stream struct {
	cfg    config
	ctx    context.Context
	cancel context.CancelFunc
	ids    atomic.Uint64

	mu     sync.Mutex
	conns  map[endpoint]*wsConn
	closed bool
}

// NewStream creates a Stream. No connection is made until the first
// subscription or WebSocket trading call.
func NewStream(opts ...Option) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	return &Stream{cfg: newConfig(opts), ctx: ctx, cancel: cancel, conns: map[endpoint]*wsConn{}}
}

// Close disconnects and closes the channels of all subscriptions.
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
	return nil
}

func (s *Stream) nextID() string { return strconv.FormatUint(s.ids.Add(1), 10) }

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
		go c.run()
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

// Subscription delivers the messages of one channel on C. C is closed after
// Unsubscribe or Stream.Close. Messages are delivered in order; a reader
// that falls behind by more than the buffer size stalls the connection, so
// consume C promptly.
type Subscription[T any] struct {
	C <-chan T

	ch     chan T
	done   chan struct{}
	mu     sync.RWMutex
	closed bool
	once   sync.Once
	decode func(Push) ([]T, error)
	conn   *wsConn
	arg    Arg
	stream *Stream
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
		s.stream.cfg.logger.Warn("okx: decode push", "channel", p.Arg.Channel, "err", err)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return
	}
	for _, it := range items {
		select {
		case s.ch <- it:
		case <-s.done:
			return
		}
	}
}

func (s *Subscription[T]) shutdown() {
	s.once.Do(func() {
		close(s.done)
		s.mu.Lock()
		s.closed = true
		close(s.ch)
		s.mu.Unlock()
	})
}

type sink interface {
	deliver(Push)
	shutdown()
}

func subscribe[T any](ctx context.Context, s *Stream, arg Arg, decode func(Push) ([]T, error)) (*Subscription[T], error) {
	ep := channelEndpoint(arg.Channel)
	if ep == endpointPrivate && s.cfg.creds == nil {
		return nil, ErrNoCredentials
	}
	c, err := s.conn(ep)
	if err != nil {
		return nil, err
	}
	ch := make(chan T, s.cfg.streamBuffer)
	sub := &Subscription[T]{C: ch, ch: ch, done: make(chan struct{}), decode: decode, conn: c, arg: arg, stream: s}
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

type endpoint string

const (
	endpointPublic   endpoint = "/ws/v5/public"
	endpointPrivate  endpoint = "/ws/v5/private"
	endpointBusiness endpoint = "/ws/v5/business"
)

var privateChannels = map[string]bool{
	"account":              true,
	"positions":            true,
	"balance_and_position": true,
	"orders":               true,
	"fills":                true,
	"liquidation-warning":  true,
	"account-greeks":       true,
}

var businessChannels = map[string]bool{
	"trades-all":      true,
	"orders-algo":     true,
	"algo-advance":    true,
	"deposit-info":    true,
	"withdrawal-info": true,
}

func channelEndpoint(channel string) endpoint {
	switch {
	case privateChannels[channel]:
		return endpointPrivate
	case businessChannels[channel],
		strings.HasPrefix(channel, "candle"),
		strings.HasPrefix(channel, "mark-price-candle"),
		strings.HasPrefix(channel, "index-candle"),
		strings.HasPrefix(channel, "sprd-"),
		strings.HasPrefix(channel, "grid-"):
		return endpointBusiness
	}
	return endpointPublic
}
