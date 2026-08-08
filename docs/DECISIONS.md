# Implementation decisions

Decisions taken while implementing the specs, and every place the specs had to
change. The specs remain the source of truth; this file records *why* they say
what they say where the reason emerged from the code rather than from design.

Entries are append-only, newest last.

---

## Phase 0 — Foundations

### `go 1.25.0` in `go.mod`, developed on 1.26.5

Require the minimum the dependencies actually need, not the toolchain that
happens to be installed. 1.25.0 is the floor imposed by `bubbletea/v2` and
`golang.org/x/sys`. Declaring 1.26 would exclude `go install` users for no gain,
and binaries are the main distribution channel anyway.

### The coverage gate needs an explicit denominator

`scripts/coverage.sh` filters `cmd/**`, `**/evals/**` and `*_gen.go` out of the
profile before measuring. A gate without a stated denominator is either
unreachable or vacuous, and the exclusions are the ones the testing convention
already justifies.

The script also prints every function below the threshold on failure. A gate
that only says "89.7%" makes you go find the gap yourself.

### CI proves the absence of cgo rather than trusting it

`CGO_ENABLED=0 go build` plus a grep for `import "C"` outside the isolated
package. Discipline decays; a check does not. This is what keeps the static
binary and cross-compilation promises from ADR-01 real.

---

## Phase 1 — Protocol

### `EventType`, `SessionState` and `ApprovalDecision` carry `Valid()`

An unknown value must fail loudly rather than fall through to a default. The
tests assert that `"yes"` and `"ALLOW"` are *not* valid decisions: the safe
reading of a typo in an approval is never "allow".

### `HTTPStatus` returns 500 for unmapped codes

A new error code that nobody added to the table surfaces as a server error
rather than silently as 200. The wrong-but-visible answer beats the
wrong-and-invisible one.

### `AsError` unwraps

Callers wrap protocol errors with context as they travel up, and the server
still needs the code to pick a status. Without `errors.As` the wrap would
silently downgrade every error to 500.

---

## Phase 2 — Context engine

### Package named `contextengine`, not `context`

It would shadow the standard library package that every other file in the
project imports. Not worth the collision for four saved characters.

### The prefix is one system message, not three

The spec orders instructions, tool definitions and summary as separate blocks.
They are emitted as a **single** system message because they form one immutable
unit, and a provider that keys its cache on message boundaries would otherwise
miss on a summary change alone. Order within the message is unchanged.

### `Assemble` rejects empty instructions

An agent with no doctrine is not a degraded agent, it is an unpredictable one.
Failing at assembly makes the misconfiguration visible at session creation
rather than as strange behavior three turns later.

### An out-of-range `Summary.UpToIdx` clamps instead of failing

It can only arrive from a corrupted session file, and refusing to assemble would
lose the whole session. Clamping degrades to "shows more history than intended",
which is recoverable.

### `Margin: 0` is treated as unset, not as a choice

**Found by a test**, not by design. A zero-valued `Config` — routine when a
config key is missing — was keeping `Margin` at zero instead of defaulting.

Zero margin is deliberately not expressible: the margin absorbs the estimation
heuristic's error, and removing it trades "compacts slightly early" for
"overruns the window". The second failure is much worse, so the ambiguity is
resolved toward safety.

### `Plan` decides, the caller summarises

Generating the summary text needs a model call. Keeping it out is what lets
`Plan` and `Assemble` stay pure and exactly golden-testable. The caller applies
the plan with `Apply`, which does not mutate the caller's history.

### Successive compactions merge summaries rather than replace

Replacing would silently erase the oldest history: the span the first summary
covered is already gone from the live window, so dropping its text loses it for
good.

### The purity guard parses imports rather than trusting review

`TestPackageImportsNothingImpure` fails the build if the package imports `time`,
`os`, `net`, `math/rand`, `syscall` or `os/exec`.

It looks like overkill until someone adds a `time.Now()` for a log line and the
prompt cache quietly stops working in production. The test costs seconds; that
investigation costs days.
