# Changelog

## Unreleased

- Fix: a response that does not decode into the Go type is no longer retried. Before, a GET request repeated it up to three times.
- Update `github.com/coder/websocket` to v1.8.15.
- Tests: every REST endpoint is checked for method, path, signing and response decoding.
- CI: test on Go 1.23 and the two supported Go releases, golangci-lint v2.15, govulncheck, Dependabot.

## v0.1.0 — 2026-10-08

First release.

- REST: market data, public data, trading statistics, account, trade, algo orders and funding account; `Client.Do` for any other endpoint.
- WebSocket: typed public and private channels, raw subscriptions, automatic ping, reconnect, re-login and re-subscription, reconnect hook.
- WebSocket trading: place, amend and cancel orders, single and batch.
- Local order book with `seqId` gap detection and resync.
- Per-endpoint client-side rate limiter, safe retries, automatic clock sync on expired timestamps.
- Iterators over history endpoints.
- Demo trading and EEA/US regions; REST on `openapi.okx.com`, WebSocket on port 443.
