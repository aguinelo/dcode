# Changelog — desktop

🇧🇷 [Versão em português](CHANGELOG.pt-BR.md)

The desktop client's own record, versioned apart from the core: tags
`desktop-vX.Y.Z`, starting at 0.1.0. Nothing has been released yet. What changed
and why, one entry each; the detail of a decision is in `docs/DECISIONS.md`.

---

## Unreleased

- **A session the window opens reads its project's configuration (N2).** The
  daemon now resolves the configuration chain in each session's workspace, so a
  session opened in a project runs under that project's `.dcode/config.toml`,
  even with the daemon started outside every project, as the desktop starts it.
  A project configuration that cannot be read refuses the session with the
  reason. `docs/DECISIONS.md` marks N2 done.
- **The visual check stays out of CI for the reason that holds.** The documents
  kept it local because the reference images were not in the repository; they
  have been since the design handoff landed. Run once on the workflow's
  `ubuntu-latest`, it scores 1.76% and 1.79% against the 1.25% limit, identical
  on four runners: Linux's Chromium hints glyphs and antialiases them in LCD
  subpixels, where macOS — on which the limit was measured — and the references
  do neither. The limit stays: loosened to fit Linux, it would leave macOS more
  than 9,000 pixels of slack, room for a missing element to pass. `AGENTS.md`,
  the workflow, the ruler's note and the root changelog stop saying the
  references are missing, and `AGENTS.md` gives the reason that holds, with the
  launch flags that brought Linux to 1.19% and 1.20% — a ruler change, left for
  a decision of its own.
- **The core now says where the daemon is (N1).** `dcode socket` prints the
  daemon's socket path — one per user, whatever the environment — or fails with
  the reason, and the connection loop asks it instead of keeping a copy of the
  core's rule. `docs/DECISIONS.md` and `docs/loop/tasks.md` name the command.
- **The next loop connects the window to a real daemon, and its ruler is written
  before the code.** `npm run check:daemon` builds the app and `dcode` from the
  checkout and runs eight scenarios, each against a fresh `dcode serve` with a
  scripted model: attaching, starting its own, saying why it could not, a new
  session with a conversation, `↵` denying an approval, `1` allowing one,
  steering a running turn, and a daemon that dies. Each is checked against what
  the daemon logged, not only against what the window drew. `docs/loop/` now
  points `/loop` at it, with the ruler protected. It fails on purpose: the window
  does not connect yet, and each scenario says where it stops. Written first
  because a loop that writes its own ruler measures what it built, not what was
  decided.
- **Seven decisions for talking to the daemon, in `docs/DECISIONS.md` (D19–D25).**
  - The desktop starts the daemon when none answers, and that daemon stops when
    the app quits.
  - One daemon serves every project, with configuration resolved per session.
  - The session list comes from the protocol.
  - `↵` denies an approval and a message typed during a turn steers it, as in the
    TUI.
  - The loop is a screen over the mechanism that exists.
  - Live tokens wait.

  Taken before any wiring, because each one changes what the wiring is. What they
  need from the core is listed as N1–N4, each a core pull request with its spec
  first.
- **The desktop area exists, apart from the core.** Electron Forge with Vite,
  TypeScript and React, in `desktop/`, with its own README, changelog, version
  and `AGENTS.md`, and a stub `go.mod` so the core's `go … ./...` stops at the
  door instead of walking into `node_modules`. Separate because the desktop
  depends on the core and never the other way round, and a client that ships on
  its own clock should not borrow the core's version.
- **The main window of the v2 design, on recorded protocol events.** The sidebar
  of projects and sessions with their state marks, one session at a time with
  its flow — the question, compact tool lines, the model's text, the delegation
  card, the activity line — the approval card with its three answers and live
  clock, the composer and the bottom bar, in both themes, following the system.
  Recorded rather than live because the window had to be right before it is
  wired: the next version connects the main process to the daemon's socket. The
  bottom bar says `gravação` so a recording is never mistaken for a daemon, and
  every action that would need one says nothing was sent.
- **A pure reducer from protocol events to what the window shows**, tested with
  vitest. It reads only fields the protocol carries, validated at the boundary: a
  payload that does not match is said in the session's flow, a repeated event
  changes nothing, a missing one is named. Where the design asks for what the
  wire does not carry — a child's progress, live tokens, the approval's requested
  paths, the loop — it draws what exists and `docs/DECISIONS.md` lists the gap,
  because an invented field is a promise the daemon never made.
- **Protocol types generated from `internal/protocol`** with tygo, committed, and
  checked for staleness in CI; every event type the core declares must have a
  reader here, and every payload shape must agree with its generated type, or
  the tests and the typecheck fail. One definition of the wire, and a desktop
  that notices when it moves.
- **Project order, labels and collapsing are local preferences**, kept in the
  browser storage of the app and never sent to the daemon: collapse and expand,
  collapse all, rename inline, move up and down, and drag to reorder. Storage
  that cannot be read or written is said, and the sidebar falls back to its
  initial arrangement.
- **The renderer cannot reach anything.** Context isolation, sandbox, no Node, a
  preload that exposes the platform and the user's name and nothing else, a
  content security policy that lets it connect nowhere, fonts bundled, no port
  opened — and lint fails on a Node import in renderer code.
- **The design can be checked in a loop.** `npm run check:visual` compares the
  window in fixture mode with the design's reference screenshots (Playwright's
  Chromium, pixelmatch) and fails past a limit whose reasoning sits next to it:
  the design's own mock, rendered by the same Chromium, is 1.15% and 1.19% from
  the references, and the window measures 1.17% and 1.19%. `desktop/docs/loop/`
  lets dcode's own `/loop` iterate against it, with the references and the ruler
  protected. Local only: the limit was measured on macOS and holds there.
