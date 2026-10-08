package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	wsPingAfter     = 20 * time.Second
	wsDeadAfter     = 30 * time.Second
	wsReplyTimeout  = 15 * time.Second
	wsWriteTimeout  = 10 * time.Second
	wsReadLimit     = 32 << 20
	wsResubscribeBy = 50
)

type wsConn struct {
	s     *Stream
	url   string
	login bool

	mu        sync.Mutex
	ws        *websocket.Conn
	changed   chan struct{}
	groups    map[string]*group
	pending   map[string]chan wsMessage
	connected bool
	lastErr   error

	lastRead atomic.Int64
	busy     atomic.Bool
}

type group struct {
	arg   Arg
	sinks map[sink]struct{}
}

type wsMessage struct {
	ID     string          `json:"id"`
	Event  string          `json:"event"`
	Op     string          `json:"op"`
	Code   string          `json:"code"`
	Msg    string          `json:"msg"`
	Arg    *Arg            `json:"arg"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	raw    []byte
}

func newWSConn(s *Stream, ep endpoint) *wsConn {
	return &wsConn{
		s:       s,
		url:     s.cfg.wsURL + string(ep),
		login:   s.cfg.creds != nil && ep != endpointPublic,
		changed: make(chan struct{}),
		groups:  map[string]*group{},
		pending: map[string]chan wsMessage{},
	}
}

func (c *wsConn) run() {
	log := c.s.cfg.logger
	for attempt := 0; ; {
		if c.s.ctx.Err() != nil {
			return
		}
		ws, err := c.connect()
		if err != nil {
			if c.s.ctx.Err() != nil {
				return
			}
			c.mu.Lock()
			c.lastErr = err
			c.signal()
			c.mu.Unlock()
			delay := reconnectDelay(attempt)
			attempt++
			log.Warn("okx: websocket connect failed", "url", c.url, "err", err, "retry_in", delay)
			select {
			case <-c.s.ctx.Done():
				return
			case <-time.After(delay):
			}
			continue
		}
		attempt = 0

		c.mu.Lock()
		reconnected := c.connected
		c.connected, c.ws, c.lastErr = true, ws, nil
		c.signal()
		args := make([]Arg, 0, len(c.groups))
		for _, g := range c.groups {
			args = append(args, g.arg)
		}
		c.mu.Unlock()

		if reconnected {
			c.resubscribe(ws, args)
			log.Info("okx: websocket reconnected", "url", c.url, "subscriptions", len(args))
			if hook := c.s.cfg.onReconnect; hook != nil {
				go hook(c.url)
			}
		}

		err = c.readLoop(ws)
		c.disconnect(ws)
		if c.s.ctx.Err() == nil {
			log.Warn("okx: websocket disconnected", "url", c.url, "err", err)
		}
	}
}

func reconnectDelay(attempt int) time.Duration {
	d := 500 * time.Millisecond << min(attempt, 6)
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d/2 + rand.N(d/2)
}

func (c *wsConn) connect() (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(c.s.ctx, wsReplyTimeout)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, c.url, &websocket.DialOptions{ //nolint:bodyclose // the library owns the handshake body
		HTTPHeader: http.Header{"User-Agent": {c.s.cfg.userAgent}},
	})
	if err != nil {
		return nil, err
	}
	ws.SetReadLimit(wsReadLimit)
	if c.login {
		if err := c.authenticate(ctx, ws); err != nil {
			_ = ws.CloseNow()
			return nil, err
		}
	}
	return ws, nil
}

func (c *wsConn) authenticate(ctx context.Context, ws *websocket.Conn) error {
	creds := c.s.cfg.creds
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	msg, _ := json.Marshal(map[string]any{
		"op": "login",
		"args": []map[string]string{{
			"apiKey":     creds.APIKey,
			"passphrase": creds.Passphrase,
			"timestamp":  ts,
			"sign":       sign(creds.SecretKey, ts+"GET/users/self/verify"),
		}},
	})
	if err := ws.Write(ctx, websocket.MessageText, msg); err != nil {
		return err
	}
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return fmt.Errorf("okx: websocket login: %w", err)
		}
		var m wsMessage
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		switch m.Event {
		case "login":
			if m.Code == "" || m.Code == "0" {
				return nil
			}
			return &APIError{Code: m.Code, Msg: m.Msg}
		case "error":
			return &APIError{Code: m.Code, Msg: m.Msg}
		}
	}
}

func (c *wsConn) readLoop(ws *websocket.Conn) error {
	c.lastRead.Store(time.Now().UnixNano())
	ctx, stop := context.WithCancel(c.s.ctx)
	defer stop()
	go c.keepAlive(ctx, ws)
	for {
		_, data, err := ws.Read(c.s.ctx)
		if err != nil {
			return err
		}
		if string(data) == "pong" {
			c.lastRead.Store(time.Now().UnixNano())
			continue
		}
		c.busy.Store(true)
		c.handle(ws, data)
		c.busy.Store(false)
		c.lastRead.Store(time.Now().UnixNano())
	}
}

func (c *wsConn) keepAlive(ctx context.Context, ws *websocket.Conn) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if c.busy.Load() {
			continue
		}
		idle := time.Since(time.Unix(0, c.lastRead.Load()))
		switch {
		case idle >= wsDeadAfter:
			_ = ws.CloseNow()
			return
		case idle >= wsPingAfter:
			wctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
			_ = ws.Write(wctx, websocket.MessageText, []byte("ping"))
			cancel()
		}
	}
}

func (c *wsConn) handle(ws *websocket.Conn, data []byte) {
	var m wsMessage
	if err := json.Unmarshal(data, &m); err != nil {
		c.s.cfg.logger.Warn("okx: malformed websocket message", "err", err)
		return
	}
	m.raw = data
	switch {
	case m.ID != "":
		c.mu.Lock()
		reply := c.pending[m.ID]
		delete(c.pending, m.ID)
		c.mu.Unlock()
		if reply != nil {
			reply <- m
		}
	case m.Event == "notice":
		c.s.cfg.logger.Info("okx: websocket notice, reconnecting", "code", m.Code, "msg", m.Msg)
		_ = ws.CloseNow()
	case m.Event == "error":
		c.s.cfg.logger.Warn("okx: websocket error", "code", m.Code, "msg", m.Msg)
	case m.Arg != nil && len(m.Data) > 0:
		c.dispatch(Push{Arg: *m.Arg, Action: m.Action, Data: m.Data})
	}
}

func (c *wsConn) dispatch(p Push) {
	c.mu.Lock()
	g := c.groups[p.Arg.key()]
	var sinks []sink
	if g != nil {
		sinks = make([]sink, 0, len(g.sinks))
		for sk := range g.sinks {
			sinks = append(sinks, sk)
		}
	}
	c.mu.Unlock()
	for _, sk := range sinks {
		sk.deliver(p)
	}
}

func (c *wsConn) disconnect(ws *websocket.Conn) {
	c.mu.Lock()
	if c.ws == ws {
		c.ws = nil
	}
	pending := c.pending
	c.pending = map[string]chan wsMessage{}
	c.mu.Unlock()
	for _, ch := range pending {
		close(ch)
	}
	_ = ws.CloseNow()
}

func (c *wsConn) shutdown() {
	c.mu.Lock()
	ws := c.ws
	groups := c.groups
	c.groups = map[string]*group{}
	c.mu.Unlock()
	if ws != nil {
		_ = ws.CloseNow()
	}
	for _, g := range groups {
		for sk := range g.sinks {
			sk.shutdown()
		}
	}
}

func (c *wsConn) resubscribe(ws *websocket.Conn, args []Arg) {
	for i := 0; i < len(args); i += wsResubscribeBy {
		batch := args[i:min(i+wsResubscribeBy, len(args))]
		msg, _ := json.Marshal(map[string]any{"id": c.s.nextID(), "op": "subscribe", "args": batch})
		ctx, cancel := context.WithTimeout(c.s.ctx, wsWriteTimeout)
		err := ws.Write(ctx, websocket.MessageText, msg)
		cancel()
		if err != nil {
			c.s.cfg.logger.Warn("okx: resubscribe failed", "url", c.url, "err", err)
			return
		}
	}
}

// signal wakes waitReady callers; c.mu must be held.
func (c *wsConn) signal() {
	close(c.changed)
	c.changed = make(chan struct{})
}

// waitReady blocks until the connection is up. Authentication failures are
// returned immediately since retrying cannot fix them.
func (c *wsConn) waitReady(ctx context.Context) (*websocket.Conn, error) {
	for {
		c.mu.Lock()
		ws, changed, lastErr := c.ws, c.changed, c.lastErr
		c.mu.Unlock()
		if ws != nil {
			return ws, nil
		}
		if errors.Is(lastErr, ErrAuth) {
			return nil, lastErr
		}
		select {
		case <-changed:
		case <-c.s.ctx.Done():
			return nil, ErrStreamClosed
		case <-ctx.Done():
			if lastErr != nil {
				return nil, errors.Join(ctx.Err(), lastErr)
			}
			return nil, ctx.Err()
		}
	}
}

// roundTrip sends a request carrying an id and waits for the reply with the
// same id.
func (c *wsConn) roundTrip(ctx context.Context, op string, args any, extra map[string]any) (wsMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, wsReplyTimeout)
	defer cancel()
	ws, err := c.waitReady(ctx)
	if err != nil {
		return wsMessage{}, err
	}
	id := c.s.nextID()
	payload := map[string]any{"id": id, "op": op, "args": args}
	for k, v := range extra {
		payload[k] = v
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		return wsMessage{}, fmt.Errorf("okx: encode %s: %w", op, err)
	}

	reply := make(chan wsMessage, 1)
	c.mu.Lock()
	c.pending[id] = reply
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := ws.Write(ctx, websocket.MessageText, msg); err != nil {
		return wsMessage{}, fmt.Errorf("okx: websocket %s: %w", op, err)
	}
	select {
	case m, ok := <-reply:
		if !ok {
			return wsMessage{}, ErrDisconnected
		}
		return m, nil
	case <-c.s.ctx.Done():
		return wsMessage{}, ErrStreamClosed
	case <-ctx.Done():
		return wsMessage{}, ctx.Err()
	}
}

func (c *wsConn) subscribeOp(ctx context.Context, op string, arg Arg) error {
	m, err := c.roundTrip(ctx, op, []Arg{arg}, nil)
	if err != nil {
		return err
	}
	if m.Event == "error" {
		return &APIError{Code: m.Code, Msg: m.Msg}
	}
	return nil
}

func (c *wsConn) add(ctx context.Context, arg Arg, sk sink) error {
	key := arg.key()
	c.mu.Lock()
	g := c.groups[key]
	isNew := g == nil
	if isNew {
		g = &group{arg: arg, sinks: map[sink]struct{}{}}
		c.groups[key] = g
	}
	g.sinks[sk] = struct{}{}
	c.mu.Unlock()
	if !isNew {
		return nil
	}

	err := c.subscribeOp(ctx, "subscribe", arg)
	if err == nil || errors.Is(err, ErrDisconnected) {
		return nil
	}
	c.mu.Lock()
	var orphans []sink
	if c.groups[key] == g {
		delete(c.groups, key)
		for other := range g.sinks {
			if other != sk {
				orphans = append(orphans, other)
			}
		}
	}
	c.mu.Unlock()
	for _, o := range orphans {
		o.shutdown()
	}
	return err
}

func (c *wsConn) remove(ctx context.Context, arg Arg, sk sink) error {
	key := arg.key()
	c.mu.Lock()
	g := c.groups[key]
	if g == nil {
		c.mu.Unlock()
		return nil
	}
	delete(g.sinks, sk)
	empty := len(g.sinks) == 0
	if empty {
		delete(c.groups, key)
	}
	connected := c.ws != nil
	c.mu.Unlock()
	if !empty || !connected {
		return nil
	}
	err := c.subscribeOp(ctx, "unsubscribe", arg)
	if errors.Is(err, ErrDisconnected) || errors.Is(err, ErrStreamClosed) {
		return nil
	}
	return err
}

func (c *wsConn) resync(ctx context.Context, arg Arg) error {
	if err := c.subscribeOp(ctx, "unsubscribe", arg); err != nil {
		return err
	}
	return c.subscribeOp(ctx, "subscribe", arg)
}
