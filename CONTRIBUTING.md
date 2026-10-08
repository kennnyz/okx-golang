# Contributing

Issues and pull requests are welcome.

- Run `go test -race ./...` and `golangci-lint run` before opening a pull request.
- New endpoints follow the existing pattern: a request struct with `json` tags, a typed response struct with `Number` for decimals and `Time` for timestamps, and an entry in the rate limit table in `ratelimit.go` with the limit from the OKX docs.
- Field names match the OKX JSON names. Check them against a real response; `integration_test.go` is the place for live checks of public endpoints.
- Never commit API keys. Live private tests read demo keys from environment variables.
