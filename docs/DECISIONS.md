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

---

## Phase 3 — Provider

### Transcript replay lives in `package provider`, not a sub-package

The spec placed it in `internal/provider/transcript/`. It implements `Transport`
and the provider tests consume it, so a sub-package would need the parent while
the parent's tests need the sub-package. Same-package avoids an interface
indirection nobody else would use.

### Overlapping model prefixes are rejected at registration, not disambiguated

The first implementation resolved ties by longest prefix. With overlaps refused
at registration, at most one family can ever claim a name — so the tie-break was
unreachable code that looked like a safety net. Removed.

Reporting the ambiguity at startup also beats resolving it cleverly at runtime:
it is a configuration bug, and startup is where configuration bugs belong.

### A stream that ends without a terminal event is a retryable transport error

Reporting success would hand the loop a half-formed turn and the model would
carry on as if the answer were complete. There is no timeout that would catch
it, so the pump synthesises the error itself.

### Credentials are redacted centrally, by value and by shape

Trusting every call site to remember is how a key ends up in a log. Registered
values are replaced, and anything shaped like a bearer token or `sk-…` is
caught even when it was never registered — a provider that echoes a key back
would otherwise leak it.

---

## Phase 4 — Policy and sandbox

### `Evaluate` takes containment as a function

Path resolution does I/O; the decision must not. Passing `inWorkspace` in keeps
the whole decision table exactly testable, which is what makes an assertion per
cell affordable — and every cell is a security decision, so sampling would have
been cheaper and exactly wrong.

### An unknown mode or policy is an error, never a default

Falling back would either surprise the user with less access than asked for or,
far worse, with more.

### Sandbox profiles canonicalise the workspace first

**Found by a test.** On macOS the temp directory is behind a symlink, so a
profile naming the unresolved path granted nothing and every write failed with
no explanation. The boundary the kernel enforces has to be the same one policy
evaluated.

This is why the sandbox tests assert the write failed *at the OS level* rather
than that a Go check returned early — that weaker assertion would have passed
with the sandbox switched off entirely.

---

## Phase 5 — Tools

### Gitignore matching is implemented here

The dedicated packages are abandoned (sabhiram 2021, denormal 2018) and go-git
would cost twenty-odd transitive dependencies to match file patterns. The
implementation covers negation, directory rules, anchoring and `**`, and errs
toward *including* a file where it differs from git: a search showing something
extra is recoverable, one silently hiding a file is not.

### Directory-only rules cover their contents

**Found by a test.** `vendor/` did not match `vendor/pkg/a.go`. The walk masked
it by skipping the whole subtree, so the rule only held because of how it was
reached — meaning of a rule must not depend on the traversal.

### A truncated read does not satisfy read-before-edit

Marking the file read after a partial read would let a later edit act on a
region the model never saw.

### `bash` declares the worst case

A shell command is opaque, so it declares any path plus network. Anything less
would let the loop schedule it in parallel with a conflicting call.

---

## Phase 6 — The loop

### Results are appended at the emission index, never at completion order

The naive channel implementation appends on completion. It passes every small
test and produces irreproducible history under load, which breaks every golden
file downstream. The test uses a deliberately slow first call to catch it.

### The concurrency note is a constant string

Interpolating timing or counts would put volatile data in the history and cost
the prompt cache on every parallel step.

### The iteration cap comes from the family

The work horizon is a property of the model. A cap sized for a ten-file refactor
truncates legitimate work on a model trained for long-horizon loops, and one
global number serves both badly.

### The repeat detector canonicalises JSON before comparing

A model that emits the same arguments with reordered keys would otherwise loop
forever while the detector saw three different calls. Unparseable input still
gets fingerprinted from its raw bytes: refusing to compare it would disable the
detector exactly when the model is producing garbage.

### A `context_size` error forces compaction and retries

It is the one case where the local estimate was wrong. Failing the turn would
throw away work the user could still have completed.

---

## Phase 7 — Wiring

### `internal/app` is the only package that reads the environment

Everything below is pure or takes its dependencies as arguments. That is what
keeps the core exactly testable and what let the end-to-end test drive real
tools against a real workspace with a scripted model.

### The session refuses to start without a sandbox

`New` fails when the mechanism cannot be established. A session that cannot
confine its own commands should not begin.

### `--yes` denies rather than allows

The flag exists for non-interactive runs. With nobody to ask, granting in
silence would be the only alternative to denying, and it is the wrong one — so
the name promises less than it appears to.

### Successful tool output is collapsed; failures expand

Failure needs attention, success needs only confirmation. `--verbose` shows
both.

### `--dump-prompt` and `--config` are the audit pair

One answers "what exactly goes to the model", the other "where did this setting
come from". Without them support conversations end at "it works on my machine".
