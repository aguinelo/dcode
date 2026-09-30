// A stub, and deliberately empty: it makes desktop/ a module of its own, so
// the core's `go vet ./...`, `go test ./...` and `go build ./...` stop at this
// directory instead of walking into node_modules. Nothing here is Go.
module github.com/aguinelo/dcode/desktop

go 1.25.0
