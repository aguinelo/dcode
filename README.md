# dcode

🇧🇷 [Versão em português](README.pt-BR.md)

![License](https://img.shields.io/badge/license-MIT-blue)
![Go](https://img.shields.io/badge/go-1.25%2B-00ADD8)
![Status](https://img.shields.io/badge/status-alpha-yellow)
![Specs](https://img.shields.io/badge/specs-10%20complete-success)
![Coverage](https://img.shields.io/badge/coverage-93%25-success)

<img src="docs/brand/mascot.svg" width="72" align="right" alt="dcode mascot">

**Dreibox Code** — an agentic coding harness for the terminal, written in Go.

> **Status: alpha.** The core is implemented and tested — the agent reads, edits,
> searches and runs commands inside an OS-enforced sandbox. There is no TUI yet and no
> released binary; build it yourself with `make build`. Expect breaking changes.

```
┌──────────────────────────────────────────────┬────────────────────────┐
│ ● dcode  MiniMax-M3  workspace-write  ctx 34%│ PLAN                   │
│                                              │                        │
│ ▸ Add CPF validation to checkout             │ ✓ 1 Map the flow       │
│                                              │ ✓ 2 Find existing check│
│   Found the flow. Validating before persist. │ ▸ 3 Implement CPF      │
│                                              │   4 Cover with tests   │
│   ⏺ read  src/checkout/handler.go   240 lines│ ⊘ 5 Run the suite      │
│   ⏺ edit  src/checkout/validate.go    +24 −2 │     └ missing dep      │
│     │ + func validateCPF(doc string) error { │                        │
│   ⏺ bash  go test ./src/checkout/  ✓ 12 pass │ 2 of 5 · 1 blocked     │
│                                              │                        │
│ › _                                          │ [p] hide               │
└──────────────────────────────────────────────┴────────────────────────┘
```

---

## Why another one

Four terminal coding agents already do this well, and each is genuinely good at
something different:

| | Language | License | Strongest at | Weakest at |
|---|---|---|---|---|
| [Claude Code](https://github.com/anthropics/claude-code) | TS + Rust | source-available | context engineering, tool design | cold start, single provider |
| [Codex CLI](https://github.com/openai/codex) | Rust | Apache-2.0 | OS-level sandboxing, governance | provider lock-in |
| [opencode](https://github.com/anomalyco/opencode) | TypeScript | MIT | 75+ providers, extensibility | runtime weight |
| [jcode](https://github.com/1jehuang/jcode) | Rust | MIT | startup latency, RAM per session | no sandbox at all |

The gap is the intersection none of them occupy: **session density _and_ a real
OS-enforced sandbox _and_ provider neutrality.** jcode has the performance and no
sandbox. Codex has the sandbox and is tied to one provider. opencode has the neutrality
and the heaviest runtime of the four.

That intersection is what this project is for.

---

## Design decisions

Five decisions, recorded before any code. Each is a load-bearing constraint on
everything that follows.

<details>
<summary><b>Go for the core</b> — chosen over Rust and TypeScript</summary>

Go gives roughly 90% of Rust's performance profile with the best CLI/TUI tooling of any
language, a concurrency model that maps directly onto the problem (N sessions, SSE
streaming, PTY multiplexing), and the deepest contributor pool in this specific domain.

The honest version: **Go and Rust scored within noise of each other.** The weighted
matrix that produced this decision separated them by single digits on a 115-point scale,
which is not enough resolution to call one correct. Go's accepted cost is GC pressure
under many concurrent sessions — which attacks exactly the thesis above, so a per-session
memory target ships as a regression test from day one.
</details>

<details>
<summary><b>Sandbox and approvals are separate concerns</b> — borrowed wholesale from Codex</summary>

- **Sandbox** is a technical boundary enforced by the kernel — `read-only`,
  `workspace-write`, `full-access`. Apple Seatbelt on macOS, bubblewrap and Landlock on
  Linux, Windows Sandbox on Windows.
- **Approval policy** is an authorization decision, orthogonal to the boundary —
  `untrusted`, `on-request`, `never`.

Keeping them separate is what reduces approval fatigue. Harnesses that conflate the two
ask too often, users disable prompting entirely, and the security model becomes
decorative. That is the real failure mode — not the sophisticated attack, but the
exhausted user.
</details>

<details>
<summary><b>Append-only context</b> — the highest-leverage performance decision</summary>

**The context prefix is never mutated between turns.** Editing anything early in the
conversation invalidates the whole KV cache and re-bills the full prompt, in both latency
and money.

Consequences most harnesses get wrong:

- MCP tool schemas are advertised at startup from cache. A server that connects on turn 5
  and injects tool definitions invalidates the cache for the entire session.
- No timestamps, token counters, or volatile state in the system prompt.
- Compaction is rare and block-wise, never incremental.
</details>

<details>
<summary><b>Client-server from the first commit</b> — cheapest now, most expensive to retrofit</summary>

The core runs as a local daemon; the TUI is just one client. Desktop apps, IDE
extensions, shared sessions and remote execution all fall out of it, and none of them fit
into a TUI monolith.
</details>

<details>
<summary><b>Provider-agnostic, with a real adaptation layer</b> — transport × family</summary>

Not just endpoint swapping. Two orthogonal axes:

- **Transport** is the wire format (`openai`, `anthropic`). Reusable, carries no
  thresholds.
- **Family** is the adaptation — system prompt, tool schema, edit strategy. Carries the
  measured behavioral thresholds and the per-model turn limits.

MiniMax M3 forced this: it speaks **both dialects**, so a single axis would mean
duplicating the whole family. The safety consequence matters more than the code shape —
*"OpenAI-compatible" describes serialization, not behavior*, so treating a wire format as
a family would apply one model's measured thresholds to another.
</details>

---

## Architecture

```mermaid
flowchart TB
    subgraph clients[Clients]
        TUI[TUI]
        IDE[IDE ext · future]
        DESK[Desktop · future]
    end

    clients -->|HTTP + SSE over unix socket| API

    subgraph daemon[dcode daemon]
        API[protocol · session · event log]
        LOOP[agent loop]
        CTX[context engine]
        BEH[behavior · prompt]
        TOOLS[tools]
        POL[policy]
        SBX[sandbox]
        PROV[provider]

        API --> LOOP
        LOOP --> CTX
        CTX --> BEH
        LOOP --> PROV
        LOOP --> TOOLS
        TOOLS --> POL
        POL --> SBX
    end

    PROV -->|transport × family| MODEL[(LLM)]
    SBX -->|seatbelt · bwrap| OS[(OS boundary)]
```

The session is an **append-only event log**. Resume, multi-client attach and session
density all fall out of that one primitive — the same principle as the model context, one
layer up.

---

## Stack

| Concern | Choice | Why |
|---|---|---|
| Language | **Go 1.25+** | see design decisions |
| TUI | `bubbletea/v2` · `lipgloss/v2` · `bubbles/v2` | best TUI tooling of any language |
| Config | `pelletier/go-toml/v2` | typed, commentable, no indentation traps |
| Sandbox | `exec` of `sandbox-exec` / `bwrap` + `golang.org/x/sys` | **no cgo**, keeps the static binary |
| gitignore | `boyter/gocodewalker` | the dedicated libs are abandoned since 2018–2021 |
| IDs | `oklog/ulid/v2` | time-ordered, filename-safe |
| Transport | stdlib `net/http` | HTTP+SSE over a unix socket needs nothing more |

Two deliberate non-choices:

- **No gRPC.** No codegen step, reachable from a future web surface, debuggable with
  `curl --unix-socket`. The bottleneck is the model, not serialization — optimizing the
  wire would be optimizing the wrong place.
- **No bundled tooling.** `grep` and `glob` are native Go. The user's own toolchain —
  tests, linters, formatters — runs through `bash` with whatever the machine already has.
  Bundling a linter would fight the project's own config.

---

## How this is built

Spec-driven development using the **RPI protocol** — four `.spec.md` files sharing a
timestamp prefix, with strict precedence:

| File | Role | Rule |
|---|---|---|
| `.r.spec.md` | Research — context, user stories, business rules | Absolute truth. If code contradicts it, the code is wrong. |
| `.p.spec.md` | Planning — schemas, contracts, types | Use exactly the names and types defined. |
| `.config.spec.md` | Config — env vars, flags, infra constants | No new env var in code without an entry here. |
| `.i.spec.md` | Implementing — ordered execution checklist | Follow the order. |

Precedence: `.r` > `.p`/`.config` > `.i`.

### The interesting part: specs for non-deterministic behavior

A harness has a problem a CRUD app does not — its most important behavior is mediated by
a language model. You cannot write this as a schema:

> when an edit fails on an ambiguous match, the agent re-reads the file instead of
> retrying blind

So every `.r.spec.md` declares which regime its scope operates in — **deterministic**,
**model-mediated**, or **mixed** — and that declaration decides how it is verified:
assertion in `go test`, or a measured threshold over fixtures.

The corollary is an architectural goal, not an accident: **push as much behavior as
possible across the line into the deterministic side.** If context assembly is a pure
function of session state, it is exactly golden-testable — and append-only context makes
that natural, because the prefix is a pure function of history.

The same principle governs where a behavior rule lives. A rule that can be enforced in
code does not belong in a prompt at all; prompt is for what cannot be structurally
enforced. And **tool error messages are a behavior surface, not diagnostics** — they are
where recovery is taught, at the only moment it is relevant, at zero cost until it
happens.

Details in [`docs/conventions/SDD-HARNESS.md`](docs/conventions/SDD-HARNESS.md).

---

## Specs

| Spec | Regime | Covers |
|---|---|---|
| [client-server-protocol](docs/specs/architecture/client-server-protocol/) | deterministic | HTTP+SSE over unix socket, event log, approval flow |
| [context-engine](docs/specs/architecture/context-engine/) | deterministic | the pure `Assemble`, compaction planning |
| [provider-adapter](docs/specs/architecture/provider-adapter/) | mixed | transport × family, error classes, retry |
| [agent-loop](docs/specs/architecture/agent-loop/) | mixed | the turn cycle, limits, parallel tools, recovery |
| [sandbox-policy](docs/specs/architecture/sandbox-policy/) | deterministic | the two orthogonal axes, OS enforcement |
| [tool-suite](docs/specs/architecture/tool-suite/) | deterministic | read, write, edit, glob, grep, bash, plan |
| [behavior-definition](docs/specs/architecture/behavior-definition/) | mixed | prompt layers, reminders, intrinsic planning |
| [configuration](docs/specs/architecture/configuration/) | deterministic | XDG layout, precedence chain, commands |
| [client-tui](docs/specs/architecture/client-tui/) | deterministic | layout, plan panel, approval modal |
| [distribution](docs/specs/architecture/distribution/) | deterministic | install, signed release, update |
| [learned-memory](docs/specs/architecture/learned-memory/) | mixed | what the agent discovers, versioned where people read it — **design only, not built** |

---

## Installing

```bash
# the install script
curl -fsSL https://raw.githubusercontent.com/aguinelo/dcode/main/install.sh | sh

# from source
go install github.com/aguinelo/dcode/cmd/dcode@latest
```

### What the install script checks

**Nothing has to be installed first.** Of rustup, bun, deno, nvm, k3s and uv, not one
requires an external verification tool, and a first install is the worst moment to ask.

**The SHA-256 always.** A failure installs nothing and leaves nothing behind.

| Source of the expected digest | Covers | Needs |
|---|---|---|
| the digest the script carries | a **substituted release** | nothing |
| `checksums.txt` from the release | a corrupted or truncated download | nothing |
| the cosign signature over that file | a substituted release | `cosign`, if you have it |

The carried digest is the one worth explaining. `checksums.txt` travels from the same host
as the tarball, so on its own it cannot catch a swapped release — whoever replaces one
replaces the other, and the pair stays consistent with itself. The digest in the script
arrived by a different route: it is committed to `main`, where a release asset can be
replaced leaving no public trace and a line in a tracked file cannot, because changing it
is a commit.

A substituted release is covered by the carried digest **or** by the signature, and either
is enough. So the script says nothing at all when either held — telling you the signature
went unchecked while the check that matters passed by a route that does not depend on it is
noise dressed as diligence. When **neither** covered it, it says so, and points at the
installer that carries this release's digests. Never at a package to install.

`DCODE_VERSION` pins a version and `DCODE_INSTALL_DIR` chooses where. An installer carries
exactly one release's digests, so a pinned install of another version is that version's own
installer: `https://github.com/aguinelo/dcode/releases/download/vX.Y.Z/install.sh`.

`dcode update` installs a newer release on request, and never on its own. It verifies the
checksum and the signature, checks that the downloaded binary actually runs, and only then
swaps it — so every failure leaves the working binary untouched. Unlike the install script
it still **requires** cosign, because it has no second route to the expected digest.

### From source, for working on dcode itself

```bash
make install          # runs the gate, then installs to ~/.local/bin
make install-fast     # skips the gate, for the edit loop
make uninstall
```

`DCODE_INSTALL_DIR` chooses somewhere else. A local build says so in its own version
string — `0.0.0-dev+a91f2c4.dirty` — because a binary that presents itself exactly like a
published release is how a bug report costs an hour finding out it was never the published
code.

For the same reason `dcode update` refuses to replace a local build: one is normally
*ahead* of the last tag, so installing the latest release would be a downgrade wearing the
word "update". `--force` lifts the refusal.

## Running it

```bash
export DCODE_API_KEY=...

dcode                              # the terminal interface
dcode "add a test for the parser"  # one task, one exit code — for scripts and CI
dcode serve                        # the daemon, for clients that outlive one terminal
dcode tui --socket /path/to.sock   # attach a client to a running daemon
```

`dcode tui` attaches to a daemon when one answers and otherwise starts its own in-process.
The client speaks the protocol either way — the embedded daemon is a deployment detail,
not a second code path — so a session that outgrows one terminal moves to `dcode serve`
without the client changing anything.

### The credential

```bash
dcode login                    # read from a prompt that does not echo
dcode login --family claude    # a second key, for another family
dcode config                   # what is stored, masked, and where from
dcode login --reveal           # print it in full, on purpose
```

The key is never taken as an argument — an argument lands in shell history and is
visible in `ps`. It is stored in the OS keychain where there is one and in a `0600`
file where there is not, one credential per model family, chosen by
`credential.backend` so that what writes and what reads always agree.
`DCODE_API_KEY` still wins over anything stored.

`config.toml` refuses anything shaped like a secret, in any section, because that
file is meant to be versioned and synced.

Two commands need no key, and they are the audit pair:

```bash
dcode --dump-prompt          # exactly what would be sent to the model
dcode --config model.name    # a setting's effective value, and where it came from
```

By default the agent runs in `workspace-write` with `on-request` approvals: it may edit
inside the workspace, and anything crossing that boundary — a write outside it, or the
network — stops and asks. Without an approver it denies, because with nobody to ask the
only alternative is granting in silence.

Inside the workspace a short list of rules asks about the things that are different in
*kind* from ordinary work — writing `.git/**` (a hook runs on the next commit, outside the
sandbox) or `.dcode/**` (the agent's own configuration), and reading a secret (which sends
it to the model provider). They are **attention, not containment**: a command pattern is
avoided by `bash -c`, and what actually contains the agent is the sandbox.

```toml
[rules]
confirm_write   = [".git/**", ".dcode/**"]
confirm_read    = [".env", "**/*.pem"]
confirm_command = ["rm -rf*"]
```

A configured list replaces the default rather than adding to it, and `dcode config
rules.confirm_write` shows what is in force and where it came from.

### Configuring it

Everything is optional; the defaults are the product. Files live under `$DCODE_HOME`, or
the XDG directories when it is unset.

```
$DCODE_HOME/
  config.toml     settings — never credentials, which come from the environment
  AGENTS.md       instructions shared with other agent tools
  DCODE.md        instructions for dcode alone; wins where they disagree
  commands/       your own /commands — markdown with frontmatter
  skills/         guidance loaded only when its trigger fires
```

A workspace carries the same set under `<workspace>/.dcode/`, and its values win. An
unknown key in `config.toml` is an error rather than a warning: a typo that is silently
ignored is the most frustrating configuration bug there is.

### Inside the interface

While the model thinks, the last few lines of its reasoning stream dimmed on
screen — the only answer to "is it going somewhere sensible". Once it acts, that
collapses to `✻ thought for 4.2s · Tab`, because on a tool-calling turn the
thinking runs five to ten times the length of the answer and would bury the
result it led to. It never enters the history the model is sent.
`behavior.show_reasoning = false` turns it off.


`/help` lists everything. `/plan` shows the plan in full, `/config <key>` answers where a
setting came from, `/model <name>` and `/clear` open a fresh session — the system prompt
is part of the prefix, and the prefix cannot be rewritten. `/init` writes DCODE.md for the
repository from what is already in it.

Typing while a turn is running queues the message; the queue drains as one turn when the
session goes idle. `^C` interrupts the turn rather than quitting. In the approval modal,
Enter denies.

---

## Roadmap

| Phase | Delivers | Status |
|---|---|---|
| **0** | `go.mod`, CI with `-race`, coverage gate, `CGO_ENABLED=0` | ✅ |
| **1** | protocol type vocabulary | ✅ 100% covered |
| **2** | context engine — the pure `Assemble` | ✅ 99% |
| **3** | provider — transport × family, both dialects | ✅ 91% |
| **4** | policy and OS sandbox | ✅ 92% / 94% |
| **5** | the seven tools | ✅ 94% |
| **6** | the agent loop | ✅ 98% |
| **7** | behaviour, config, wiring, CLI | ✅ 91% |
| **8** | event log, unix socket, SSE, reference client | ✅ 95% / 92% / 93% |
| **9** | TUI client, commands, skills, reminders, distribution | ✅ 95% / 96% / 92% |

The aggregate gate is **95%** and every package clears its own 90% floor; the suite runs under `-race` on macOS and Linux.

Beyond MVP: multiple providers, MCP, plugins, session sharing, desktop, IDE.

**Self-hosting milestone.** A pull request to dcode written end to end by dcode, passing
review and the coverage gate with no manual edits. It is the best eval the project has: its
own test suite and review checklist become the fitness function. Its bias mitigation is
mandatory — keep a non-Go codebase in the eval fixtures, or the agent gets excellent at
Go and mediocre elsewhere without the metric noticing.

---

## Repository layout

```
docs/
  conventions/            bilingual — X.md is English, X.pt-BR.md is Portuguese
    LANGUAGE.md           the bilingual policy itself
    SDD-HARNESS.md        applying spec-driven development to a harness
    TESTING.md            TDD, bug reproduction rule, coverage gate
    GO-CODE-REVIEW.md     Go review checklist, with product-specific checks
  brand/                  bilingual — mascot, logomark, palette, voxel maps
  specs/                  Portuguese only — see LANGUAGE.md section 3
    architecture/         cross-cutting specs
    domains/              feature specs
```

```
internal/
  protocol/       the wire vocabulary, no logic and no I/O
  contextengine/  the pure Assemble — where ADR-03 lives or dies
  provider/       transport × family, replay transport, credential redaction
  policy/         the two orthogonal axes, pure decision table
  sandbox/        seatbelt and bubblewrap, driven as binaries
  tools/          read write edit glob grep bash plan
  loop/           the turn cycle
  behavior/       the prompt builder
  config/         roots, precedence chain, config.toml, commands, instructions
  session/        the append-only event log, approvals, the session manager
  server/         the daemon: unix socket, protocol routes, SSE
  tui/            the terminal client — a pure reducer and a pure renderer
  update/         signature and checksum verification, atomic binary swap
  app/            the only package that reads the environment
pkg/client/       the reference client, and the first consumer of the protocol
cmd/dcode/        argument parsing and printing
install.sh        verifies before it installs, or installs nothing
```

---

## Contributing

Too early for code contributions — the specs are still moving.

What is useful right now is argument. If a design decision above looks wrong, open an
issue and say why. The reasoning is written down precisely so it can be attacked: every
decision has a stated cost, and the Go-versus-Rust call in particular was close enough
that new information could flip it.

### Workflow

**GitHub Flow.** `main` is always deployable. Work happens on short-lived branches cut
from `main` and returns through a pull request.

```
main ──┬─────────────────────────┬──▶
       └── feat/event-log ── PR ─┘
```

Branch names follow the change type: `feat/`, `fix/`, `docs/`, `chore/`, `refactor/`. For
spec-driven work, use the spec slug without its timestamp — `feat/client-server-protocol`.

**[Conventional Commits](https://www.conventionalcommits.org/)** on every commit message
and PR title. Breaking changes carry `!` before the colon and explain the break in the
body. For anything marked `stable` in a `.p.spec.md`, a breaking change also requires a
`changelog/` entry and a major version bump.

Commits that change technical behavior must keep the corresponding spec in sync — a spec
is never allowed to go stale relative to the code.

**Authorship.** Commits carry a single author and no co-author trailers. Every commit is
attributable to one person; tooling that assisted does not get a byline.

### Testing

**TDD.** Test first, watch it fail, then write the code. A test that has never been seen
red is not a safety net.

**Every bug gets a reproducing test — before the fix.** Reproduce it in a failing test,
confirm it fails for the reported symptom, then fix it, and the same test passes
unmodified. A `fix:` PR without a new test is blocked. Regression tests are permanent.

**95% aggregate coverage gate, 90% per package**, with an explicit denominator: deterministic code in `internal/` and
`pkg/`. Generated code, `main` wiring, and model-mediated paths are excluded — the last
because it cannot be verified by assertion at all, only by measured thresholds over
fixtures. That exclusion is deliberate pressure in the right direction.

The gate is a floor, not a target. A test that exercises a line without asserting anything
is a review finding even when coverage is green.

Full rules in [`docs/conventions/TESTING.md`](docs/conventions/TESTING.md).

### Language

This project is bilingual. English is canonical and takes the bare filename; Portuguese
carries a `.pt-BR` suffix. The README and everything under `docs/conventions/` exist in
both, cross-linked at the top.

Two deliberate exceptions: **specs are Portuguese only**, because RPI makes `.r.spec.md`
the absolute truth and that rule needs exactly one source of truth — a drifted spec is
worse than a missing one because it still looks authoritative. **Commits and code comments
are English only**, because changelog tooling assumes a single language.

Full policy in [`docs/conventions/LANGUAGE.md`](docs/conventions/LANGUAGE.md).

---

## Brand

The mascot is three stacked boxes; the logomark is a **D** built from the same three
boxes. Its eye is the `⏺` that marks every execution line in the TUI, so the mark repeats
itself on screen instead of being applied on top.

Designed as three pieces that stack — the name becomes the object, and it prints without
supports. Files, palette and voxel maps in [`docs/brand/`](docs/brand/).

## License

[MIT](LICENSE) — the same license as opencode and jcode.
