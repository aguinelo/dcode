# DCode Box — the dcode desktop

🇧🇷 [Versão em português](README.pt-BR.md)

The desktop client of dcode, named **DCode Box** (D65; the command is still
`dcode`): a second client of the same daemon the terminal client talks to, in
Electron, so it looks the same on macOS, Linux and Windows. macOS first.

> **Status.** 0.1.0, unreleased. The window runs on a real `dcode serve`: it
> attaches to the one answering on the socket, or starts one. It opens on the
> **Crew** (D63): the daemon's conversations, live and ended, grouped by
> project and searchable on the left; the conversation in the middle; and on
> the right what it established — Mudanças (files, the git diff, the PR and its
> CI), Pronto (the seal and the criteria) and Sessão (folder, branch, model,
> mode, tokens). An orchestrator's conversation adds an **Agentes** tab, the
> live list of its children (D64). The **grid** of attention (D30, D31) stays
> one switch away in the top bar: several conversations side by side, each one
> worked in its own window. From either one a session is driven end to end — a
> new one in a chosen folder, a turn with images and documents, a correction, a
> stop, an approval with the TUI's answers, a model or mode change, undo,
> compact, a command run in the terminal below — and a `/loop` is started,
> followed and stopped (D54–D56). ⌘K finds any conversation by its title or
> project. Dark by default, light on choice (D35), in the mascot's orange
> (D49, D52). In a browser it shows a recording instead, and the window says
> `gravação`.

## Running it

```bash
cd desktop
npm ci
npm start             # the app, with the renderer on Vite's dev server
npm run dev:renderer  # the renderer alone, in a browser, in fixture mode
npm run build         # a production build of main, preload and renderer into .vite/
```

`npm start` downloads Electron's binary on first use. The window looks for the
daemon on `DCODE_SOCKET`, else where `dcode socket` says, and attaches to it; with
nothing answering there it starts `dcode serve` — from `DCODE_BIN`, else
`~/.local/bin/dcode`, else the `PATH` — and stops it when the app quits. A daemon
it did not start keeps running. The installed `dcode` must be 0.23.0 or later,
which serves the list of conversations. In a browser, `?fixture=` opens one
of the recorded states of `src/fixtures/scenes.ts`, with the window controls
drawn where macOS would put them: `crew`, `crew-vazia`, `crew-loop`,
`crew-orquestra` and `crew-orquestra-vazia` on the Crew, and
`02-janela-principal-rodando` and `03-janela-principal-aprovacao` on the grid's
maximized window.

## How it is built

- **Main process** (`src/main/`) owns the window and is the daemon's client: HTTP
  and the event streams over its Unix socket, the daemon it started, and the
  question before quitting while a session works. The window keeps the native
  macOS controls inside the sidebar's title strip.
- **Preload** (`src/preload/`) hands the renderer a narrow API — one named
  channel per request and per notice, in `src/shared/api.ts` — and nothing else.
- **Renderer** (`src/renderer/`, React) never sees Node, the filesystem or the
  socket: context isolation, sandbox, and a content security policy that lets it
  connect nowhere. Geist, Geist Mono and Fragment Mono are bundled; nothing
  loads from the network.
- **Protocol types** (`src/protocol/generated.ts`) are generated from the core's
  `internal/protocol` with tygo — `npm run gen:protocol` — committed, and checked
  for staleness in CI. The desktop reads the core; the core never reads the
  desktop.
- **State** (`src/state/`) is a pure reducer from protocol events to what the
  window shows, per session, plus the derivations the sidebar and the bottom bar
  need. It reads only fields the protocol carries; where the design asks for more,
  `docs/DECISIONS.md` lists the gap.

## The design loop

The window is measured against reference screenshots of the approved look, so
it can be iterated without drifting from it — by a person, or by dcode.

```bash
npm run check:visual
npm run check:visual -- --refs /path/to/refs/design/desktop/screenshots
```

It builds the renderer, opens each reference state in Playwright's Chromium at
1600×960 (reduced motion, fixed clock; the grid's two states in the dark, the
Crew's in both themes), compares it with the PNG of
the same name in `refs/design/desktop/screenshots/` using pixelmatch, and writes
the screenshot, the diff and `report.json` to `test-results/visual/`. It prints
one line per state and fails when any is past the limit. The states, the limit
and the reasoning behind it live in `tests/visual/states.json`. Chromium for
Playwright is installed once with `npx playwright install chromium`.

Whether the window talks to a daemon is measured the same way:

```bash
npm run check:daemon
npm run check:daemon -- steers
```

It builds the app and `dcode` from this checkout and runs each scenario in
`tests/daemon/` against a fresh `dcode serve` with a scripted model, launching
the app as Electron and driving it by clicks and keys. Each scenario is checked
against what the daemon logged, not only against what the window drew. Failures
leave a screenshot and the logs in `test-results/daemon/`.

To let dcode iterate on it, run from the repository root:

```
dcode
/loop desktop/docs/loop
```

The loop works until `desktop/docs/loop/done.toml` is met — typecheck, lint,
tests, the protocol types, the visual check and the daemon check. It treats as
protected the reference images, `tests/visual/`, `tests/daemon/` and the loop
folder: they are the ruler, and changing them is not progress. What the loop is
building now, and why it needs a session with full access, is in
`docs/loop/tasks.md`.

**The references are captures of the approved window** (since 2026-10-13; the
handoff's mock before that): 02 and 03 of the grid, and since 2026-10-17 05 to
12 of the Crew, each in both themes. A new look is new PNGs in
`refs/design/desktop/screenshots/`, in a PR of its own after it is approved —
never in the PR that changes the window. Measure the new references' own floor
before trusting the old limit (see `tests/visual/states.json` and D18).

## More

- [`AGENTS.md`](AGENTS.md) — commands, conventions, what CI runs
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — why the window does what it does, and the protocol gaps
- [`CHANGELOG.md`](CHANGELOG.md) — this area's own record and version
