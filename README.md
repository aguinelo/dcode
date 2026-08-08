# harness

An agentic coding harness for the terminal, written in Go.

> **Status: specification phase.** There is no implementation yet — this repository currently
> contains architecture decisions and specs. Nothing here is installable or runnable.
> Starring it means you're interested in where it goes, not that it does anything today.

---

## Why another one

Four terminal coding agents already do this well, and each one is genuinely good at
something different:

| | Language | License | Strongest at | Weakest at |
|---|---|---|---|---|
| [Claude Code](https://github.com/anthropics/claude-code) | TS + Rust | source-available | context engineering, tool design | cold start, single provider |
| [Codex CLI](https://github.com/openai/codex) | Rust | Apache-2.0 | OS-level sandboxing, governance | provider lock-in |
| [opencode](https://github.com/anomalyco/opencode) | TypeScript | MIT | 75+ providers, extensibility | runtime weight |
| [jcode](https://github.com/1jehuang/jcode) | Rust | MIT | startup latency, RAM per session | no sandbox at all |

The gap is the intersection none of them occupy: **session density _and_ a real
OS-enforced sandbox _and_ provider neutrality.** jcode has the performance and no
sandbox. Codex has the sandbox and is tied to one provider. opencode has the
neutrality and the heaviest runtime of the four.

That intersection is what this project is for.

---

## Design decisions

Five decisions, recorded before any code. Each one is a load-bearing constraint on
everything that follows.

### Go for the core

Chosen over Rust and TypeScript. Go gives roughly 90% of Rust's performance profile
with the best CLI/TUI tooling of any language, a concurrency model that maps directly
onto the problem (N sessions, SSE streaming, PTY multiplexing), and the deepest
contributor pool in this specific domain.

The honest version: **Go and Rust scored within noise of each other.** The weighted
matrix that produced this decision separated them by single digits on a 115-point
scale, which is not enough resolution to call one correct. Go's accepted cost is GC
pressure under many concurrent sessions — which attacks exactly the thesis above, so
a per-session memory target ships as a regression test from day one.

### Sandbox and approvals are separate concerns

Borrowed wholesale from Codex, because it is the best-designed model of the four.

- **Sandbox** is a technical boundary enforced by the kernel — `read-only`,
  `workspace-write`, `full-access`. Apple Seatbelt on macOS, Landlock and bubblewrap
  on Linux, Windows Sandbox on Windows.
- **Approval policy** is an authorization decision, orthogonal to the boundary —
  `untrusted`, `on-request`, `never`.

Keeping them separate is what reduces approval fatigue. Harnesses that conflate the
two ask too often, users disable prompting entirely, and the security model becomes
decorative.

### Append-only context

The single highest-leverage performance decision. **The context prefix is never
mutated between turns.** Editing anything early in the conversation invalidates the
whole KV cache and re-bills the full prompt, in both latency and money.

Consequences that most harnesses get wrong:

- MCP tool schemas are advertised at startup from cache. A server that connects on
  turn 5 and injects tool definitions invalidates the cache for the entire session.
- No timestamps, token counters, or volatile state in the system prompt.
- Compaction is rare and block-wise, never incremental.

### Client-server from the first commit

The core runs as a local daemon; the TUI is just one client. This is the decision
that is cheapest now and most expensive to retrofit — desktop apps, IDE extensions,
shared sessions and remote execution all fall out of it, and none of them fit into a
TUI monolith.

### Provider-agnostic, with a real adaptation layer

Not just endpoint swapping. Provider-agnostic harnesses lose to tuned ones on the same
model because system prompts, tool schemas and edit strategies need to be adapted per
model *family*. That adaptation layer is budgeted work, not configuration.

---

## How this is built

Spec-driven development, using the **RPI protocol** — four `.spec.md` files sharing a
timestamp prefix, with a strict precedence order:

| File | Role | Rule |
|---|---|---|
| `.r.spec.md` | Research — context, user stories, business rules | Absolute truth. If code contradicts it, the code is wrong. |
| `.p.spec.md` | Planning — schemas, contracts, types | Use exactly the names and types defined. |
| `.config.spec.md` | Config — env vars, flags, infra constants | No new env var in code without an entry here. |
| `.i.spec.md` | Implementing — ordered execution checklist | Follow the order. |

Precedence: `.r` > `.p`/`.config` > `.i`.

### The interesting part: specs for non-deterministic behavior

A harness has a problem a CRUD app does not — its most important behavior is mediated
by a language model. You cannot write this as a schema:

> when an edit fails on an ambiguous match, the agent re-reads the file instead of
> retrying blind

So every `.r.spec.md` here declares which regime its scope operates in — deterministic,
model-mediated, or mixed — and that declaration decides how it gets verified:
assertion in `go test`, or a measured threshold over fixtures.

The corollary is an architectural goal, not an accident: **push as much behavior as
possible across the line into the deterministic side.** If context assembly is a pure
function of session state, it is exactly golden-testable — and append-only context
makes that natural, because the prefix is a pure function of history.

Details in [`docs/conventions/SDD-HARNESS.md`](docs/conventions/SDD-HARNESS.md).

---

## Repository layout

```
docs/
  conventions/
    SDD-HARNESS.md        how to apply spec-driven development to a harness
    GO-CODE-REVIEW.md     Go review checklist, including product-specific checks
  specs/
    architecture/         cross-cutting specs (protocol, context, sandbox)
    domains/              feature specs
```

Implementation will live in `internal/` (everything that is not a public contract) and
`pkg/` (everything that is).

---

## Current status

| Area | State |
|---|---|
| Architecture decisions | recorded |
| Client-server protocol spec | written, `experimental` |
| Context engine spec | not started |
| Sandbox spec | not started |
| Provider adapter spec | not started |
| Implementation | **not started** |

The first implementation milestone is the protocol type vocabulary — no server, no
I/O, just the shared types and their round-trip tests.

---

## Contributing

Too early for code contributions — the specs are still moving.

What is useful right now is argument. If a design decision above looks wrong, open an
issue and say why. The reasoning is written down precisely so it can be attacked:
every decision has a stated cost, and the Go-versus-Rust call in particular was close
enough that new information could flip it.

### Workflow

**GitHub Flow.** `main` is always deployable. Work happens on short-lived branches cut
from `main` and returns through a pull request.

```
main ──┬─────────────────────────┬──▶
       └── feat/event-log ── PR ─┘
```

Branch names follow the change type: `feat/`, `fix/`, `docs/`, `chore/`, `refactor/`.
For work driven by a spec, use the spec slug without its timestamp —
`feat/client-server-protocol`.

**[Conventional Commits](https://www.conventionalcommits.org/).** Every commit message
and every PR title. The type prefix is what drives changelog generation and
communicates blast radius at a glance.

```
feat:     new capability
fix:      bug fix
docs:     documentation only
refactor: behavior-preserving change
perf:     performance, no behavior change
test:     tests only
chore:    build, tooling, dependencies
```

Breaking changes carry a `!` before the colon (`feat!:`) and explain the break in the
body. For anything marked `stable` in a `.p.spec.md`, a breaking change also requires
a `changelog/` entry and a major version bump.

Commits that change technical behavior must keep the corresponding spec in sync —
a spec is never allowed to go stale relative to the code.

### Testing

**TDD.** Test first, watch it fail, then write the code. A test that has never been
seen red is not a safety net.

**Every bug gets a reproducing test — before the fix.** Reproduce the bug in a failing
test, confirm it fails for the reported symptom, then fix it, and the same test passes
unmodified. A `fix:` PR without a new test is blocked. Regression tests are permanent
and never simplified away.

**90% coverage gate**, with an explicit denominator: deterministic code in `internal/`
and `pkg/`. Generated code, `main` wiring, and model-mediated paths are excluded — the
last one because it cannot be verified by assertion at all, only by measured thresholds
over fixtures.

That exclusion is deliberate pressure in the right direction: behavior on the
deterministic side of the line counts toward coverage and is exactly verifiable, so the
incentive is to keep moving behavior across it.

The gate is a floor, not a target. It catches files with no tests; it does not prove
correctness — and a test that exercises a line without asserting anything is a review
finding even when coverage is green.

Full rules in [`docs/conventions/TESTING.md`](docs/conventions/TESTING.md).

### Language

Specs are written in Portuguese (project convention); code, comments, commit messages
and this README are in English.

---

## License

[MIT](LICENSE) — the same license as opencode and jcode.
