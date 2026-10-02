# DCode desktop

🇧🇷 [Versão em português](README.pt-BR.md)

The desktop client of dcode: a second client of the same daemon the terminal
client talks to, in Electron, so it looks the same on macOS, Linux and Windows.
macOS first.

> **Status.** 0.1.0, unreleased. The main window of the v2 design — the sidebar
> of projects and conversations, one session at a time, the approval card, the
> composer and the bottom bar — runs on a real `dcode serve`: it attaches to the
> one answering on the socket, or starts one. Its sidebar is the daemon's list of
> conversations, live and ended; a session is driven from the window — a new one
> in a chosen folder, a turn, a correction, a stop, an answer to an approval — and
> ⌘K finds any conversation by its title or project. In a browser it shows a
> recording instead, and the bottom bar says `gravação`.

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
which serves the list of conversations. In a browser,
`?fixture=02-janela-principal-rodando` or `?fixture=03-janela-principal-aprovacao`
opens one of the design's reference states, with the window controls drawn where
macOS would put them.

## How it is built

- **Main process** (`src/main/`) owns the window and is the daemon's client: HTTP
  and the event streams over its Unix socket, the daemon it started, and the
  question before quitting while a session works. The window keeps the native
  macOS controls inside the sidebar's title strip.
- **Preload** (`src/preload/`) hands the renderer a narrow API — one named
  channel per request and per notice, in `src/shared/api.ts` — and nothing else.
- **Renderer** (`src/renderer/`, React) never sees Node, the filesystem or the
  socket: context isolation, sandbox, and a content security policy that lets it
  connect nowhere. Geist and Geist Mono are bundled; nothing loads from the
  network.
- **Protocol types** (`src/protocol/generated.ts`) are generated from the core's
  `internal/protocol` with tygo — `npm run gen:protocol` — committed, and checked
  for staleness in CI. The desktop reads the core; the core never reads the
  desktop.
- **State** (`src/state/`) is a pure reducer from protocol events to what the
  window shows, per session, plus the derivations the sidebar and the bottom bar
  need. It reads only fields the protocol carries; where the design asks for more,
  `docs/DECISIONS.md` lists the gap.

## The design loop

The window is measured against the design's own screenshots, so it can be
iterated until it matches — by a person, or by dcode.

```bash
npm run check:visual
npm run check:visual -- --refs /path/to/refs/design/desktop/screenshots
```

It builds the renderer, opens each reference state in Playwright's Chromium at
1600×960 (dark theme, reduced motion, fixed clock), compares it with the PNG of
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

**A new Claude Design version** is new reference PNGs in
`refs/design/desktop/screenshots/`. The loop then shows the distance to them.
Measure the new design's own floor before trusting the old limit (see
`tests/visual/states.json`).

## More

- [`AGENTS.md`](AGENTS.md) — commands, conventions, what CI runs
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — why the window does what it does, and the protocol gaps
- [`CHANGELOG.md`](CHANGELOG.md) — this area's own record and version
