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

---

## Phase 8 — Sessions, the daemon and the client

### The event log is the session, and the session is server-side

A client holds one number: the last sequence it saw. Everything else — history,
plan, pending approvals — lives in the daemon. That is what makes reattaching
indistinguishable from having watched the session live, and it is why the TUI
can be killed mid-turn without losing anything.

### `dcode tui` embeds a daemon when none is running

The client always speaks the protocol; whether the daemon is in this process or
another is a deployment detail, not a second code path. A single binary that
just works was not worth a second, untested execution path — so there is only
one, and `dcode serve` merely moves it.

The embedded daemon binds a private socket rather than the default one. Two
terminals opened without a shared daemon would otherwise race to bind the same
address, and the loser would fail to start for a reason the user cannot act on.

### The socket path is deliberately short

A Unix socket is capped near 104 bytes on macOS, and the XDG state directory
alone can exhaust that. `$XDG_RUNTIME_DIR/dcode.sock` when it exists, otherwise
`$TMPDIR/dcode-<uid>.sock` — the uid keeps two users on one machine apart.

### The workspace is validated where it enters

`CreateSessionRequest.Workspace` is the one field a remote client fully
controls, and it anchors every boundary the session will enforce. A path that is
not an existing directory is refused at creation with `workspace_invalid`,
rather than at the first tool call — by which point the user has started waiting
and the failure looks like the tool's rather than the request's.

### Each session gets its own sandbox, resolver and provider

A session is the unit of confinement. Sharing any of it across sessions would
let one workspace's boundary apply to another.

### `/clear` opens a new session rather than wiping the view

Context is server-side and append-only, so there is no way to unsay something to
the model. Clearing the screen while the model still remembers would be a lie
about what it knows.

`/model` works the same way and for the same reason: the system prompt is part
of the prefix, and the prefix cannot be rewritten (ADR-03).

---

## Phase 9 — Configuration, behaviour channels and distribution

### `config.toml` is parsed by a small strict reader, not a TOML library

The documented file is sections and scalar values. A full parser is both a
dependency and a much larger surface than the file needs, and every construct
outside the subset — arrays, inline tables, arrays of tables — is rejected by
name rather than ignored. A user writing valid TOML that dcode does not support
is told so instead of watching it do nothing.

### `KnownKeys` is the schema, and the key-to-variable mapping is bijective

One key, one environment variable, in both directions, asserted by a test. It is
what lets `--config <key>` name a single origin, and it makes a key that exists
in the file but not in the environment impossible to introduce by accident.

The credential check runs *before* the unknown-key check, on the key's own name.
An unknown section is exactly where a secret would otherwise slip through.

### `sandbox.policy` was renamed to `sandbox.approval_policy`

The internal resolution key now matches the name the user writes in the file.
Two names for one setting is how `--config` starts answering a question nobody
asked.

### The parallel-execution note moved into the reminder channel

It was loose text appended to the tool results. The behaviour spec puts it in
the reminder channel, which is where it belongs: appended, never prefixed,
constant per kind, and marked so the model does not read it as the user
speaking.

### `Message.Reminder` marks the channel on the wire

Reminders ride the user role because that is the only channel every provider
accepts mid-conversation. The flag is what lets a client refuse to render one as
something the user said, and the `<system-reminder>` wrapper is what lets the
model tell them apart.

### Skill triggers are matched deterministically

An explicit `triggers:` list is matched as a phrase; without one, the
`when_to_use` line is matched on its significant words and two distinct hits are
required. A single common word would drag a skill into a task that merely
mentioned it in passing. The determinism is what keeps a replayed session
byte-identical to the live one — whether the model then *uses* the loaded body
is the model-mediated part, and that is what the eval threshold measures.

### `when_to_use` is capped at 120 characters

The index line is paid for on every turn of every session. A skill that
describes itself in a paragraph charges every session for a context most of them
never enter. A skill without the line is refused outright: unindexed, the model
never learns it exists.

### Built-in commands beat user commands, and the shadowing is reported

The moment a user file could redefine `/config`, no advice about dcode would be
true of any particular installation. The collision is reported as a note rather
than swallowed, because the override that did not happen is exactly what would
otherwise be spent an afternoon on.

### `Expand` does no I/O and runs nothing

A command that could read a file or run a process would be a second, undeclared
tool surface with none of the sandbox or approval machinery pointed at it. The
boundary would be bypassed by a markdown file.

### `/init` is a turn, not a template

What belongs in DCODE.md depends on what the repository already says about
itself, and only reading it can answer that.

### The updater fails closed when it cannot verify

No cosign, no install. "Installed, but unverified" is the worst of both worlds:
the user ends up with a binary and the impression that it went fine. Shelling
out to cosign rather than linking sigstore means the verification path is the
same one a user can run by hand to reproduce the result.

### `Apply` stages beside the target and checks the binary runs first

The staging directory sits next to the binary so the final step is a rename
within one filesystem — across filesystems a rename is a copy, and an
interrupted copy leaves a machine with no working dcode on it. Running
`--version` on the candidate before the swap is what stops a working binary
being replaced by one that does not run on that machine.

### The version notice never enters the model's context

`CheckedAt` changes between turns. In the prefix it would invalidate the whole
cached prompt for the sake of a line the user reads once (ADR-03). A network
failure is silent by contract: the stale answer stands, and no exit code
changes.

### Fixed: `wrap` did not break a word wider than its column

A path or stack frame longer than the plan panel was emitted whole, and the
terminal wrapped it — which destroys the fixed layout the panel exists to hold.
Found by the width invariant test, reproduced by
`TestWrapBreaksAWordWiderThanTheColumn`, then fixed. The regression test asserts
both that no line exceeds the column and that breaking loses no characters.

---

## Phase 10 — What the first real model call found

Everything before this was verified against recorded transcripts. The first turn
against a live MiniMax-M3 found two defects in minutes, and both were in the gap
between frames — the one place a hand-written fixture never looks.

### A frame is not a unit of meaning

`Family.Decode(ev, tools)` claimed decoding was a pure function of one event.
The wire disagrees: a tool call's name arrives in one frame and its arguments in
the next. The decoder emitted on the first fragment, where `arguments` is still
`""`, so every tool ran with no input and answered "field is required" — three
times over, until the repeat guard ended the turn having done nothing.

`Decode` became `NewDecoder(tools) Decoder`, stateful and single-use, returning
zero or more events per frame. Both were forced: zero when a frame carried only
a fragment, several when the end of a stream flushes the calls it assembled.

The assembly lives in the family rather than in the pump because *how* a call is
split is dialect-specific — `index` within `tool_calls` for OpenAI, content
blocks with `input_json_delta` for Anthropic. Putting it in the pump would make
the pump know both formats, which is exactly the coupling the transport × family
split exists to prevent.

The Anthropic decoder had the same defect in its own shape and was fixed with
it, before anything ran against that dialect.

### The flush has to be idempotent

MiniMax repeats `finish_reason` — twice in the captured stream. Emitting on each
would run every tool a second time.

### Reasoning is not the answer

M3 sends its thinking twice: once in `reasoning`, once in `content` wrapped in
`<think>` markers. Reading `content` printed the thinking to the user and, worse,
appended it to the history as the assistant's own words, where it would be paid
for on every later turn and read back as something it had said out loud.

A frame carrying `reasoning` is a thinking frame. Its `content` is not an answer.

Reasoning is dropped by the loop rather than forwarded: there is no protocol
event for it, and inventing one so a client can show thinking is a feature, not
part of fixing the leak. `EventReasoningDelta` stops at the provider boundary so
the information exists when that feature is wanted.

### A frame of pure framing produces nothing

The model closes its reasoning with `\n</think>\n\n</think>` — markers and
newlines, nothing else. Stripping the markers alone left three blank lines at the
top of every answer, so a frame that was only framing is dropped whole. Content
with no marker keeps its whitespace: that is how a paragraph break arrives.

### The fixtures were not realistic, and that is what hid this

Two tests fed a single-frame stream with no terminator — something the wire never
produces. Under the corrected decoder they failed, because validation now happens
where it belongs: at the end of the stream. The fix was to make the fixtures
terminate the way a real stream does.

The real capture is now committed at
`internal/provider/testdata/minimax-m3-toolcall.sse`. It is worth more than the
tests written against it: every hand-made approximation of this stream was
passing.

### Fixed: the plan panel was unreachable on an ordinary terminal

Reported from use: the panel never appeared. It was two things at once.

The panel hides itself below 100 columns, which is a sound default — but a
standard terminal is 80, so most people would never see it. Worse, `p` only
flipped a `PanelHidden` boolean, and the width check ran *after* it, so pressing
the key on an 80-column terminal did nothing. The feature was unreachable
exactly where someone would go looking for the key.

The spec already said the right thing — RN-5: "the user's explicit preference
takes precedence over the automatic one in both directions." The code was not
implementing it, so no rule changed and there is no changelog entry; the
contract gained the detail that makes the rule expressible.

A boolean cannot express it. From `auto` there is no flag to flip, so the mode
is three-valued: auto, shown, hidden. Responsiveness answers the case where the
user never noticed the window got narrow; a keypress is the user noticing.

Two smaller consequences, both found by writing the failing test first:

- **The panel narrows before it disappears.** Its width is capped at a quarter
  of the screen with a floor of 16. At 80 columns that trades four panel cells
  for four stream cells, which is the right way round — the panel holds short
  lines and the stream holds diffs.
- **A collapsed panel announces itself.** The status bar carries the counter and
  the key that brings it back. Collapsed in silence is indistinguishable from
  broken, and the key was documented only inside the panel that was not on
  screen.

---

## Phase 11 — The interface

Three things the user asked for — scrolling, a processing indicator, navigation
— turned out to be mostly debt: the spec already promised `PgUp`/`PgDn`, `Esc`,
`?` and an animated indicator, and none of them existed. The gap between spec
and code was the work.

### Rendering everything, then taking a window from it

The renderer only ever produced the tail, which is why scrolling was impossible:
there was nothing above the screen to scroll back to. `StreamLines` now renders
the whole stream and `Window` takes the visible slice, both pure over the model.
Scroll position is clamped rather than trusted, because content grows underneath
it — a position valid one event ago can be past the end now.

Following is the default and stops the moment the user scrolls up. Reading
something while the stream pushes it off the screen is the single most
irritating thing a live log can do. It resumes at the bottom, because that is
what going there means.

### Fixed: a letter was a shortcut

`p` toggled the panel on an empty line, so typing "primeiro" produced "rimeiro".
Found by a test that typed a word rather than a character. The rule that came
out of it — a letter is never a shortcut on a line the user types into — is now
RN-16, and the toggle is `Ctrl+P`.

`?` stays, as punctuation and as the convention every pager already uses.

### Elapsed time is the client's, the tool's duration is the daemon's

The turn timer must advance between events, and a server timestamp is only right
at the instant it arrives — so the client measures it. A tool's duration is the
opposite: the client cannot see when execution began, only when the events
arrived, so the daemon measures it and sends it. It is measured around `Execute`
alone, because the wait for an approval is the user's time and folding it in
would make every gated call look slow.

### Colour is roles, and monochrome emits nothing

The palette maps roles, not colours: a caller asking for "red" has already
decided something the theme should decide. Two invariants make it removable —
styling never changes measured width, and a disabled palette emits no escapes at
all, not even a reset. The second was a real defect: `clipStyled` appended a
reset unconditionally, so a `NO_COLOR` user got escape bytes in plain output and
every width measurement counted them.

### Summaries come from metadata, not from prose

`read → 240 lines`, `edit → +24 −2`, `bash → exit 1` are in the spec and were
being approximated from the first line of output. The tools now report what they
did, and the protocol carries it. Rebuilding those numbers by matching text
breaks the day the wording changes — in every client at once.

### The usage was never arriving, for two separate reasons

First, the OpenAI dialect reports `"usage": null` on every frame unless the
request opts in with `stream_options.include_usage`. Every provider speaking
that dialect is silent by default.

Second, and only visible once the first was fixed: MiniMax attaches the usage to
a **final frame that repeats `finish_reason`**, and never sends `[DONE]`.
Terminating on the first finish threw the accounting away. `Decoder` gained
`Close()` so the terminal event waits until the transport says nothing more is
coming — while a stream that stops mid-message still reports a truncation rather
than a clean finish.

### The context meter disappeared on a large window

M3's window is a million tokens, so five thousand tokens is zero percent in
integer division and the meter never showed — precisely early in a long session,
when it is most wanted. Below one percent it now says `ctx <1%` rather than
rounding to nothing.

---

## Phase 12 — Adopting the design handoff

The handoff in `docs/design_handoff_dcode_tui/` was itself derived from what we
had already defined, so most of it was confirmation. The value was in three
things it added and one thing it got wrong.

### The diff is for the human, so it costs no tokens

`edit` returned prose and the client showed that prose. The spec had said since
day one that the diff is what gets reviewed.

Tools now produce a unified diff that travels on the event and **never enters
the history**: the model wrote the edit and already knows what changed, so a
400-line diff in `Output` would be paid for on every subsequent turn of the
session, for a reader who does not need it. Putting it beside the output rather
than in it is what makes it free.

Shown without being asked for, because it is the thing being reviewed — but as a
preview, with `Tab` revealing the rest. The cut states how much is hidden and
which key shows it; "truncated" alone leaves the reader unable to judge whether
it matters.

The diff is a small LCS with the common head and tail trimmed first, which is
what keeps a one-line change in a large file from paying for the whole file. Two
files with nothing in common degrade to a plain replacement rather than stalling
the turn in a quadratic table.

### Two orders, and they are not the same one

The status bar reads model-then-mode and drops model-before-everything. Writing
that as one list is how the mode ends up being what disappears — which is
exactly what the handoff's own narrow-terminal mock did, dropping the sandbox
indicator two sections after stating that it is the one field where being wrong
is dangerous.

We adopted the mock's layout and inverted what it gives up. The mode is not in
the drop order at all; at 40 columns the bar is `✓ dcode  workspace-write` and
nothing else.

The first attempt had the loop iterating the wrong way and added the *least*
important field first — caught by the test that asserts the model name goes
before the plan counter.

### The menu is a consequence, not a mode

Command completion is derived from the line on every keystroke rather than
toggled open and closed. A menu with its own state drifts out of step with the
text the moment anything else edits the line. `Esc` suppresses it for the line as
it is, and any edit revives it — the first version only revived on insertion, so
backspacing left it dead, which a test caught.

While open the menu owns four keys and no others. `Enter` in particular still
sends: the menu must not swallow the one key that matters most.

### Not adopted: the bubbles components

The handoff states purity as a principle and then recommends `bubbles/viewport`
and `textinput`, which hold their own mutable state. Our viewport and input are
pure functions over the model, which is why forty tests for scrolling and line
editing run with no terminal. The README says the spec wins where they diverge,
and the spec is the purity.

### Fixed: bash asked for consent to a crossing the sandbox already prevented

Reported from use: every `bash` command stopped and asked, including `go build`.

The tool declared `Network: true` unconditionally — "a shell command is opaque,
declaring the worst case is the only honest answer". The reasoning was right and
the conclusion was wrong, because the worst case is bounded by the sandbox that
will run the command, and outside `full-access` a confining sandbox is
guaranteed to exist (`BackendNone` is refused in every other mode).

Measured before changing anything:

| Configuration | Write outside the workspace | Network |
|---|---|---|
| `workspace-write`, `allow_network=false` | blocked by the OS | blocked by the OS |
| `workspace-write`, `allow_network=true` | blocked by the OS | reachable |

So with the default configuration the prompt claimed a network crossing that
could not happen. Approving granted nothing — the command still could not
resolve a host — and denying stopped the whole command rather than its network
access. The user was answering a question that was not the one on screen, which
is worse than not asking: it teaches that the prompt means something it does not.

`Bash.Declare` now declares network only when the sandbox was built to permit
it. The consequences line up with what each mode already promises:

- `workspace-write` with the network shut: no prompt, and the command is still
  confined to the workspace with no network — by the OS, which is what was
  holding the line all along.
- `workspace-write` with the network open: every command asks, and now the
  question is true — the approval really is the only thing in the way.
- `read-only`: unchanged, still denied.
- `full-access`: unchanged, nothing is asked.

The end-to-end test that asserted "a network crossing must be refused when
nobody approves it" was kept and moved to the configuration where a crossing
exists. A second test pins the other half: with the network shut, nothing is
refused *and* nothing gets through.

What did not change: a shell command still declares that it writes, in every
configuration. That is the part the sandbox cannot decide for us, because
writing inside the workspace is exactly what `workspace-write` grants.

---

## Phase 10 — Measuring the behavioural contracts

### The eval package is `internal/evals`, not `internal/provider/evals`

The spec placed it under the provider. It cannot go there: a scenario needs a
real transport, the HTTP transport lives in `internal/app`, and `app` imports
`provider` — so an eval package under `provider` closes the cycle.

`internal/evals` is a leaf that nothing imports, which is what lets it reach
both. It is also where the contracts of every other spec will land, so the
scenarios of one product live in one place rather than scattered by whichever
package happens to be measured.

### `Measure` sits outside the build tag

It takes the attempt as a function and reaches no model, so it belongs in the
ordinary suite. Deciding whether a threshold was met is exactly the judgement
that must not itself depend on a paid, flaky, once-in-a-while measurement.

What stays behind the tag is only the three scenarios, because only they open a
connection.

### A failure to measure is not a verdict

`Attempt` returns `(bool, error)` and the two are different questions. The bool
is what the model did; the error means the run never happened. A result with
any error is `Sound() == false` and can never be `Met()`, however good the rate
of the runs that did complete.

Collapsing them is the failure mode that matters: a flaky network would read as
a behavioural regression, someone would go looking for it in the prompt, and
the threshold would eventually be lowered to make the noise stop.

### Errored runs stay in the denominator

`Rate` divides by the runs attempted, not by the runs that completed. Dividing
by completions would let a scenario that failed nineteen times out of twenty
report the one success as 100%.

### The adapter validates declaration and JSON, never the schema

Found while writing `toolcall-schema-valid`. `validateToolCall` checks that the
tool name was declared and that the arguments parse as JSON; the tool's schema
is not applied. The spec said "já validado contra schema" and now says what the
code does.

The consequence is concrete: the judge for that scenario decodes the nested
structure itself and asserts the required fields, because trusting the adapter
would have measured "the model returned JSON" — a question `{}` answers.

`no-phantom-tool` is unaffected and its 100% threshold remains legitimate: the
declared-name check is real, and that is precisely the thing the threshold
measures.

### The non-session escape hatch grew a second directory, not a weaker rule

`TestNonSessionKeysAreReadSomewhere` proved a key was read by grepping
`cmd/dcode`. The eval keys are read by `internal/evals`, so the guard now reads
both directories — and still fails if a key is excused and spelled differently
where it is consumed, which is the property that caught `update.channel`.

Widening the search beats excusing the keys outright: an escape hatch nobody
checks is the hole the whole file exists to close.

### `make eval-build` exists because a tagged test rots in silence

Nothing in `make check` compiles a file behind the `eval` tag, so the scenarios
would drift out of sync with the code they measure and nobody would learn until
the next paid run. `go vet -tags eval` costs a second and makes that impossible.
