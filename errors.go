package okx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrNoCredentials       = errors.New("okx: credentials required for private endpoint")
	ErrRateLimited         = errors.New("okx: rate limited")
	ErrAuth                = errors.New("okx: authentication failed")
	ErrTimestampExpired    = errors.New("okx: request timestamp expired")
	ErrInvalidParameter    = errors.New("okx: invalid parameter")
	ErrInsufficientBalance = errors.New("okx: insufficient balance")
	ErrOrderNotFound       = errors.New("okx: order does not exist")
	ErrServiceUnavailable  = errors.New("okx: service unavailable")
	ErrEmptyResponse       = errors.New("okx: empty response")
)

var codeSentinels = map[string]error{
	"50011": ErrRateLimited,
	"50040": ErrRateLimited,
	"50061": ErrRateLimited,
	"50001": ErrServiceUnavailable,
	"50004": ErrServiceUnavailable,
	"50013": ErrServiceUnavailable,
	"50026": ErrServiceUnavailable,
	"50102": ErrTimestampExpired,
	"50112": ErrTimestampExpired,
	"50100": ErrAuth,
	"50101": ErrAuth,
	"50103": ErrAuth,
	"50104": ErrAuth,
	"50105": ErrAuth,
	"50110": ErrAuth,
	"50111": ErrAuth,
	"50113": ErrAuth,
	"50119": ErrAuth,
	"50120": ErrAuth,
	"60009": ErrAuth,
	"60024": ErrAuth,
	"50014": ErrInvalidParameter,
	"51000": ErrInvalidParameter,
	"51001": ErrInvalidParameter,
	"51008": ErrInsufficientBalance,
	"51131": ErrInsufficientBalance,
	"51603": ErrOrderNotFound,
}

// APIError is a rejected request. Code is the top-level OKX code; for order
// operations the per-item codes are in Items and usually explain more.
type APIError struct {
	HTTPStatus int
	Code       string
	Msg        string
	Items      []ItemError
}

// ItemError is the result of one item in a (batch) operation.
type ItemError struct {
	Code string
	Msg  string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "okx: code %s", e.Code)
	if e.Msg != "" {
		b.WriteString(": " + e.Msg)
	}
	for _, it := range e.Items {
		fmt.Fprintf(&b, " [%s: %s]", it.Code, it.Msg)
	}
	if e.HTTPStatus != 0 && e.HTTPStatus != http.StatusOK {
		fmt.Fprintf(&b, " (http %d)", e.HTTPStatus)
	}
	return b.String()
}

// Is matches sentinel errors such as ErrRateLimited against the top-level
// code, every item code and the HTTP status.
func (e *APIError) Is(target error) bool {
	if codeSentinels[e.Code] == target {
		return true
	}
	for _, it := range e.Items {
		if codeSentinels[it.Code] == target {
			return true
		}
	}
	switch e.HTTPStatus {
	case http.StatusTooManyRequests:
		return target == ErrRateLimited
	case http.StatusUnauthorized:
		return target == ErrAuth
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return target == ErrServiceUnavailable
	}
	return false
}

// HasCode reports whether err is an APIError carrying code at the top level
// or in any item.
func HasCode(err error, code string) bool {
	var e *APIError
	if !errors.As(err, &e) {
		return false
	}
	if e.Code == code {
		return true
	}
	for _, it := range e.Items {
		if it.Code == code {
			return true
		}
	}
	return false
}
