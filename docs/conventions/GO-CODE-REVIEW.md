# Code review checklist — Go (CLI / harness / daemon)

🇧🇷 [Versão em português](GO-CODE-REVIEW.pt-BR.md)

Convention of this repository. Tests: `go test -race ./...`. Main build: `CGO_ENABLED=0`.
Shared directories: `internal/**`, `pkg/**`.

Reusable, high-hit-rate checks. Not exhaustive; apply what fits the diff.

## Concurrency (the largest source of bugs in Go)

- **Ownerless goroutine:** every `go func()` needs a clear termination path. Launched
  without a `context`, `WaitGroup` or stop channel → leak. Ask: who cancels this?
- **`context` as the first parameter:** a function doing I/O without a `ctx` is not
  cancelable. In the agent turn path that becomes a session that survives Ctrl-C →
  **blocker**.
- **Ignored `ctx`:** receiving `ctx` and never checking `ctx.Done()` nor passing it down
  is worse than not receiving it — it gives a false guarantee.
- **Loop variable capture:** fixed in Go 1.22 and the `go` directive is 1.25, so this
  never applies here. Kept as a note because contributors arriving from older codebases
  still flag it — and flagging a non-issue costs review time.
- **Unbuffered channel on a write path:** the producer blocks if the consumer is gone. In
  event streaming, always `select` with `ctx.Done()` on send, never a bare `ch <- x`.
- **Copied `sync.Mutex`:** a struct with a mutex passed by value. `go vet` catches it —
  confirm it runs in CI.
- **Data race:** a change introducing shared state without a lock requires `go test -race`
  in CI. If the pull request touches concurrency and CI does not run `-race`, that is a
  process finding.

## Errors

- **Swallowed `err`:** `_ = f()` or `if err != nil { return nil }` without context. In Go
  the error is the contract — discarding it is information loss.
- **Wrapping:** `fmt.Errorf("...: %w", err)` to preserve the chain. Without `%w`,
  `errors.Is`/`errors.As` stop working upstream.
- **Sentinel vs string:** comparing errors via `strings.Contains(err.Error(), ...)` is
  **always** a finding. It must be `errors.Is` against an exported sentinel.
- **`panic` in a library:** acceptable only for an unrecoverable programming error at
  initialization. In a request or turn path → blocker.
- **`recover` without re-logging:** silently swallowing a panic hides a structural bug.

## Resources and I/O

- **`defer` inside a loop:** `defer f.Close()` within `for` only runs at function exit —
  descriptors accumulate. Extract to a function or close explicitly.
- **Unclosed HTTP body:** `resp.Body.Close()` is mandatory even on error, and **read to
  EOF** before closing to reuse the connection.
- **Missing timeout:** `http.Client{}` without `Timeout` never gives up. Every external
  call (provider, MCP) needs a timeout **and** a `ctx`.
- **`io.ReadAll` on untrusted input:** a provider response or tool output without a limit
  → OOM. Use `io.LimitReader`.

## Specific to this product

- **Context prefix mutation:** code that edits, reorders or removes an already-sent
  message violates append-only and invalidates the KV cache → **blocker**.
- **Timestamp or counter in the system prompt:** same consequence — invalidates the cache
  every turn.
- **Late-assembled tool schema:** a tool definition that only exists after connecting to
  MCP at runtime invalidates the prefix. It must come from a startup cache.
- **Context assembly must be a pure function:** `(session state) → []Message`, with no
  I/O and no clock inside. That is what makes exact golden testing possible — a side
  effect there is an architecture finding, not a style one.
- **Hot-path allocation:** allocating per token or per event becomes GC pressure under
  swarm. If the pull request touches the turn loop, ask: does this allocate per delta?
  Look for buffer reuse.
- **`exec.Command` without policy:** execution outside the boundary defined in the
  permission spec → **blocker**. It must go through the policy-aware executor, never a
  bare `exec`.
- **cgo in the core:** `import "C"` outside the build-tag-isolated package breaks the
  static binary and cross-compilation. CI validates `CGO_ENABLED=0` on the main build.

## SOLID / DRY / structure

- **Interface defined by the producer:** in Go, interfaces belong to the **consumer**. A
  package exporting an interface alongside its single implementation is usually premature
  abstraction.
- **Wide interface:** more than 3–4 methods usually signals mixed responsibility. Prefer
  small interfaces and composition.
- **`internal/` vs `pkg/`:** anything that is not a public contract **must** live in
  `internal/`. An exported type in `pkg/` becomes a compatibility commitment — confirm it
  was intentional and that the `.p.spec.md` declares its stability level.
- **`any` in a public API:** discards the typing that is the reason to use Go. Generics or
  a concrete type almost always fit.
- **Global dependency:** a package-level `var db *sql.DB`, an implicit singleton, an
  `init()` with side effects → hurts testability and hides initialization order.

## Tests

- **Table-driven:** it is the idiom of the language. A run of copy-pasted `t.Run` calls
  with identical bodies → refactor into a table.
- **Golden files:** a change to serialized output (event, assembled context, TUI render)
  without an updated `testdata/` means missing coverage.
- **`t.Parallel()` with shared state:** a parallel test writing to a package variable or
  the same temp directory → flake.
- **Tests depending on the network or a real model:** behind a build tag or
  `testing.Short()`. Model-mediated behavior is measured by threshold in the behavioral
  contracts section of the relevant `.p.spec.md`, **not** in the deterministic `go test`
  run.
