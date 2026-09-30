# DCode desktop

🇧🇷 [Versão em português](README.pt-BR.md)

The desktop client of dcode: a second client of the same daemon the terminal
client talks to, in Electron, so it looks the same on macOS, Linux and Windows.
macOS first.

> **Status.** 0.1.0, unreleased. This version draws the main window of the v2
> design — the sidebar of projects and sessions, one session at a time, the
> approval card, the composer and the bottom bar — from **recorded** protocol
> events. It does not connect to a daemon yet; the bottom bar says `gravação`
> (recording) so nobody mistakes it for a live one. Wiring the main process to
> the daemon's socket is the next version.

## Running it

```bash
cd desktop
npm ci
npm start             # the app, with the renderer on Vite's dev server
npm run dev:renderer  # the renderer alone, in a browser, in fixture mode
npm run build         # a production build of main, preload and renderer into .vite/
```

`npm start` downloads Electron's binary on first use. The window opens on the
recording. In a browser, `?fixture=02-janela-principal-rodando` or
`?fixture=03-janela-principal-aprovacao` opens one of the design's reference
states, with the window controls drawn where macOS would put them.

## How it is built

- **Main process** (`src/main/`) owns the window and, next version, the daemon's
  Unix socket. The window keeps the native macOS controls inside the sidebar's
  title strip.
- **Preload** (`src/preload/`) hands the renderer a narrow API — the platform and
  who the user is — and nothing else.
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

To let dcode iterate on it, run from the repository root:

```
dcode
/loop desktop/docs/loop
```

The loop works until `desktop/docs/loop/done.toml` is met — typecheck, tests and
the visual check — and treats the reference images and `tests/visual/` as
protected: they are the ruler, and changing them is not progress.

**A new Claude Design version** is new reference PNGs in
`refs/design/desktop/screenshots/`. The loop then shows the distance to them.
Measure the new design's own floor before trusting the old limit (see
`tests/visual/states.json`).

## More

- [`AGENTS.md`](AGENTS.md) — commands, conventions, what CI runs
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — why the window does what it does, and the protocol gaps
- [`CHANGELOG.md`](CHANGELOG.md) — this area's own record and version
