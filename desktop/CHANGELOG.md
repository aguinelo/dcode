# Changelog — desktop

🇧🇷 [Versão em português](CHANGELOG.pt-BR.md)

The desktop client's own record, versioned apart from the core: tags
`desktop-vX.Y.Z`, starting at 0.1.0. Nothing has been released yet. What changed
and why, one entry each; the detail of a decision is in `docs/DECISIONS.md`.

---

## Unreleased

- **A conversation that ends is no longer reported as a failure.** When a
  session closes, the daemon ends its stream and the list shows it ended; the
  window took the stream's end for a fault and said the events had stopped
  coming. It now waits a moment for the list, which arrives over a connection of
  its own, and says so only when the conversation is still live there.
- **The core lists the models a session can ask for.** One route answers, for a
  session's workspace, the model it gets by default and each profile it can
  switch to, each with its family and whether that family has measurements
  behind it — what the per-session model menu will read, instead of reading
  `models.toml` and keeping a copy of the core's list of measured families. The
  protocol types are regenerated.
- **A turn that goes round again speaks in a new message.** When the done check
  sends a turn back to work, the model's next words came out glued to its last
  ones — "…o parser.O teste…" — because streamed text joined any open message of
  the same turn. The daemon reports each new round with a progress event; text
  now joins only the message of the same turn and the same round.
- **⌘K finds any conversation.** The search the design draws, over the daemon's
  list: every word typed must be in the title or the project, without case or
  accents; the conversations running or waiting for you first, then one group per
  project; `tab` for the scope, the arrows for the cursor, `esc` to clear and then
  to close. `↵` opens the one under the cursor as a click on its row would, so an
  ended conversation is continued. The preview shows what the list carries — the
  state, the seal, the workspace, branch, model, turns and last event — and
  `docs/DECISIONS.md` records what the design asks for and the list does not
  (D27, L17).
- **The window runs on a real daemon.** Inside Electron, the main process
  attaches to the `dcode serve` answering on the socket — `DCODE_SOCKET`, else
  what `dcode socket` prints — or starts one there as its child, from
  `DCODE_BIN`, `~/.local/bin/dcode` or the `PATH`; it stops only the one it
  started, and asks first when quitting would stop a session that is running or
  waiting for you (D19). The bottom bar says connecting, connected with the
  daemon's version, lost or failed, with the reason. The sidebar is the daemon's
  list of conversations (N3), live and ended, by project: a live one opens its
  stream from the first event and resumes from the last when it drops; an ended
  one is continued in a new session the window opens (D21). The field sends a
  turn, corrects a running one (D23) and stops it; an approval is answered with
  `1`, `2`, `3`, `esc` and `↵`, which denies (D22); Nova sessão and ⌘N ask for a
  folder and open a session there. Every refusal shows the daemon's message, and
  nothing sent to a daemon that died looks accepted. The renderer still sees
  neither Node nor the socket: the preload hands it one named channel per request
  and per notice (`src/shared/api.ts`). A browser, or `?fixture=`, still shows
  the recording, and the visual check reads 1.20% and 1.21% — the numbers D22 and
  D23 were measured at.
- **The app keeps its data in a folder of its own.** Electron's default is the
  product name in the system's data folder, `~/Library/Application Support/DCode`
  on macOS — and the file system there does not tell `DCode` from `dcode`, the
  folder dcode keeps its configuration, records and state in. Every `npm start`
  wrote Chromium's storage, caches and cookies beside `models.toml`, and deleting
  the app's data would have taken dcode's configuration with it. The app now uses
  `dcode-desktop` there, and `DCODE_DESKTOP_USER_DATA` names another, absolute
  folder — which the check already passed and the app ignored. What earlier runs
  left in dcode's folder is Chromium's and can be deleted by hand; dcode's own
  entries there are `doctrine`, `models.toml`, `sessions` and
  `update-check.json`. `docs/DECISIONS.md` D26.
- **The loop now takes the window all the way to usable.** With N1–N3 in the
  core, `docs/loop/tasks.md` adds what day-to-day work needs on top of the
  connection: the sidebar listing every conversation, live and ended, from the
  core's list and its one stream; opening an ended one by continuing it in a new
  session; and ⌘K finding any conversation by its title. The ruler gains a
  scenario for each — `lists-recorded`, `continues-recorded`, `searches` — so
  `check:daemon` has eleven. Their daemon side was proven against a real
  `dcode serve` before they were committed; they fail on purpose, at the window,
  until it connects.
- **The core lists every conversation for the sidebar (N3).** One route with the
  live and the recorded conversations, each with title, project, state, branch,
  model, turns, seal, diff and last activity, and one stream of what changes —
  what the sidebar and the search will read instead of one stream per session.
  The protocol types are regenerated, and `docs/DECISIONS.md` marks N3 done.
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
