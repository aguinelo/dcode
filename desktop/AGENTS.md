# dcode desktop — for agents

The Electron client of the dcode daemon. Everything in the repository root's
`AGENTS.md` applies here too — stage named paths only, no `Co-Authored-By`,
every failure explicit, the changelog on the branch that makes the change, CI
read before a push is called finished. This file adds only what is specific to
`desktop/`.

## Commands

Run from `desktop/`.

```bash
npm ci                  # install exactly what the lockfile says
npm start               # the app (Electron, renderer on Vite's dev server)
npm run dev:renderer    # the renderer alone in a browser, in fixture mode
npm run build           # production build of main, preload and renderer into .vite/
npm run typecheck       # tsc on the renderer project and on the Node project
npm run lint            # ESLint, flat config
npm test                # vitest: the pure state fed protocol events, and the main process's logic
npm run gen:protocol    # regenerate src/protocol/generated.ts from internal/protocol (needs Go)
npm run check:protocol  # regenerate, and fail if the committed file was stale
npm run check:visual    # the window against the design's screenshots (see below)
npm run check:daemon    # the window against a real dcode serve and a scripted model (see below)
npm run gen:icons       # the app icon's sizes (.icns, .ico, .png) from assets/icon/icon.svg (D49)
```

`npm run build` does not package or sign; there are no makers yet.

## What CI runs

`.github/workflows/desktop.yml`, on pull requests and pushes to `main` that touch
`desktop/**`, `internal/protocol/**` or the workflow: `npm ci`, typecheck, lint,
tests, the production build, and the protocol-types staleness check.

**`check:visual` is not in CI**, on purpose: its limit is known to hold only on
macOS. The references in `refs/design/desktop/screenshots/` are captures of the
approved window made by this check itself, with Playwright's Chromium on macOS
(D18), and on that machine the window scores 0.00% against them. How far the
same Chromium on the workflow's `ubuntu-latest` lands from these captures has
never been measured. What is known is older and about other references: against
the handoff's PNGs, before the captures, Linux scored 1.76% and 1.79% where
macOS scored 1.17% and 1.19%, because there Chromium hints the glyphs and antialiases them in LCD
subpixels where macOS draws them unhinted in grey — the layout was the same and
the difference was in the text. So a Linux run is expected to be off by the
text alone, by an amount nobody has measured against the 1.25% in
`tests/visual/states.json`; putting it in CI would be a gate whose margin is a
guess. Measuring it, and either fixing the font rendering
(`--disable-lcd-text --font-render-hinting=none` once narrowed the gap) or
capturing Linux references, changes the ruler `docs/loop/done.toml` protects,
so it is a decision of its own. Until then, run it on macOS. Chromium for
Playwright is installed once with
`npx playwright install chromium`; `--refs <dir>` or `DCODE_DESIGN_SCREENSHOTS`
point it at other references. A missing reference or browser fails it with the
reason; it never skips.

**`check:daemon` is not in CI** either: it launches the app as Electron, which
needs a display, and builds `dcode` from the checkout with Go (`DCODE_BIN` names
a binary instead). Run it locally; a missing Go, git or display fails it with the
reason.

## Conventions

- **The renderer never sees Node.** No `electron` or Node import under
  `src/renderer/`, `src/state/`, `src/protocol/`, `src/fixtures/` or
  `src/shared/` — lint fails on one. What the renderer needs from outside comes
  through the preload's `DcodeApi` (`src/shared/api.ts`), one named request at a
  time.
- **Only what the protocol carries.** State is folded from the generated types in
  `src/protocol/generated.ts`, never edited by hand. A field the design wants and
  the wire does not carry is a gap: draw what exists, and add it to
  `docs/DECISIONS.md` under "Lacunas". Never invent a field, and never parse a
  tool's output text to rebuild a number.
- **A new event type in the core fails the tests here** until `src/protocol/events.ts`
  reads it; a changed payload field fails the typecheck until its `Shape` agrees.
- **The state is pure.** `src/state/` takes events and `now` as arguments and
  reads no clock, storage or DOM; the tests in `tests/unit/` feed it events.
- **Interface text is Portuguese**, the design's words, in `src/renderer/text.ts`
  and the components. Code, comments and commit messages are English.
- **Decisions about the interface** go to `docs/DECISIONS.md`, in Portuguese. No
  SDD specs for them; protocol changes keep the core's spec discipline.
- **Files stay under 500 lines**, CSS included. Generated files — the protocol
  types and the lockfile — are the exception.
- **Versioning is this area's own.** `CHANGELOG.md` and `CHANGELOG.pt-BR.md` here,
  tags `desktop-vX.Y.Z` from 0.1.0. The root changelog only records root-level
  changes (this area existing, its workflow).

## The loop

`docs/loop/` is what `/loop` works toward now: `tasks.md` says what to build and
`done.toml` says when it is done. It has two rulers, and `done.toml` protects
both, with the reference images and the loop folder itself — a change there
changes the measurement and has to be made deliberately, never to make a run
pass.

- `tests/visual/states.json` is the visual ruler: which states are measured, how,
  and the limit with the reasoning that chose it. The references are captures
  of the approved window — 02 and 03 of the grid, 05 to 12 of the Crew in both
  themes — and their floor is 0.00% on macOS. When new references arrive,
  measure their own floor on the same Chromium before trusting the old limit.
- `tests/daemon/` is the daemon ruler: the scenarios, the scripted model, and the
  attributes the window exposes for them (`docs/loop/tasks.md`, "O que a régua
  lê"). A scenario checks the window against what the daemon logged, so a window
  that only looks right does not pass.
