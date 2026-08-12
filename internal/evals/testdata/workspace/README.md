# example

A small Go service that summarises rows and reports what it counted.

## Layout

- `stats.go` — the `Summary` type and its counters
- `internal/config` — configuration, resolved in layers
- `internal/legacy` — older parsing code, with its own conventions
- `internal/version` — the version this binary reports

## Running the tests

    go test ./...
