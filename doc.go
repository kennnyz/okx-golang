// Package okx is a client for the OKX v5 API: REST, WebSocket push channels
// and WebSocket trading.
//
// Create a Client for request/response calls and a Stream for real-time
// data. Both take the same options:
//
//	opts := []okx.Option{
//		okx.WithCredentials(apiKey, secretKey, passphrase),
//		okx.WithDemoTrading(),
//	}
//	client := okx.NewClient(opts...)
//	stream := okx.NewStream(opts...)
//
// Prices and sizes are Number, a string type that preserves the exact
// decimal OKX sends. Timestamps are Time, which decodes OKX's Unix
// millisecond strings into time.Time.
//
// Errors returned by OKX are *APIError and match sentinel errors such as
// ErrRateLimited or ErrInsufficientBalance with errors.Is.
package okx
