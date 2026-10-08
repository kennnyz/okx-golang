# Changelog

## v0.1.0 — 2026-10-08

First release.

- REST: market data, public data, trading statistics, account, trade, algo orders and funding account; `Client.Do` for any other endpoint.
- WebSocket: typed public and private channels, raw subscriptions, automatic ping, reconnect, re-login and re-subscription, reconnect hook.
- WebSocket trading: place, amend and cancel orders, single and batch.
- Local order book with `seqId` gap detection and resync.
- Per-endpoint client-side rate limiter, safe retries, automatic clock sync on expired timestamps.
- Iterators over history endpoints.
- Demo trading and EEA/US regions; REST on `openapi.okx.com`, WebSocket on port 443.
