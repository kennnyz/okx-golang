package okx

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// PlaceOrder places an order over the private WebSocket connection, which
// is faster than REST and shares the same rate limits.
func (s *Stream) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (OrderResult, error) {
	return wsOne(ctx, s, "order", []PlaceOrderRequest{req})
}

// PlaceOrders places up to 20 orders; check Err on each result.
func (s *Stream) PlaceOrders(ctx context.Context, reqs []PlaceOrderRequest) ([]OrderResult, error) {
	return wsBatch(ctx, s, "batch-orders", reqs)
}

func (s *Stream) CancelOrder(ctx context.Context, req CancelOrderRequest) (OrderResult, error) {
	return wsOne(ctx, s, "cancel-order", []CancelOrderRequest{req})
}

// CancelOrders cancels up to 20 orders; check Err on each result.
func (s *Stream) CancelOrders(ctx context.Context, reqs []CancelOrderRequest) ([]OrderResult, error) {
	return wsBatch(ctx, s, "batch-cancel-orders", reqs)
}

func (s *Stream) AmendOrder(ctx context.Context, req AmendOrderRequest) (OrderResult, error) {
	return wsOne(ctx, s, "amend-order", []AmendOrderRequest{req})
}

// AmendOrders amends up to 20 orders; check Err on each result.
func (s *Stream) AmendOrders(ctx context.Context, reqs []AmendOrderRequest) ([]OrderResult, error) {
	return wsBatch(ctx, s, "batch-amend-orders", reqs)
}

func wsOne[T any](ctx context.Context, s *Stream, op string, args []T) (OrderResult, error) {
	res, err := wsTrade(ctx, s, op, args, false)
	if err != nil {
		return OrderResult{}, err
	}
	if len(res) == 0 {
		return OrderResult{}, ErrEmptyResponse
	}
	return res[0], nil
}

func wsBatch[T any](ctx context.Context, s *Stream, op string, args []T) ([]OrderResult, error) {
	return wsTrade(ctx, s, op, args, true)
}

func wsTrade[T any](ctx context.Context, s *Stream, op string, args []T, batch bool) ([]OrderResult, error) {
	if s.cfg.creds == nil {
		return nil, ErrNoCredentials
	}
	c, err := s.conn(endpointPrivate)
	if err != nil {
		return nil, err
	}
	var extra map[string]any
	if s.cfg.requestTTL > 0 {
		extra = map[string]any{"expTime": strconv.FormatInt(time.Now().Add(s.cfg.requestTTL).UnixMilli(), 10)}
	}
	m, err := c.roundTrip(ctx, op, args, extra)
	if err != nil {
		return nil, err
	}
	var out []OrderResult
	if err := decodeResponse(http.StatusOK, m.raw, batch, &out); err != nil {
		return nil, err
	}
	return out, nil
}
