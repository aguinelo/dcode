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
npm test                # vitest: the pure state, fed protocol events
npm run gen:protocol    # regenerate src/protocol/generated.ts from internal/protocol (needs Go)
npm run check:protocol  # regenerate, and fail if the committed file was stale
npm run check:visual    # the window against the design's screenshots (see below)
```

`npm run build` does not package or sign; there are no makers yet.

## What CI runs

`.github/workflows/desktop.yml`, on pull requests and pushes to `main` that touch
`desktop/**`, `internal/protocol/**` or the workflow: `npm ci`, typecheck, lint,
tests, the production build, and the protocol-types staleness check.

**`check:visual` is not in CI**, on purpose: the reference PNGs live in
`refs/design/desktop/screenshots/`, which is not in the repository yet. Run it
locally, pointing at them with `--refs <dir>` or `DCODE_DESIGN_SCREENSHOTS` when
they are not at the repo-relative default. It needs Chromium for Playwright once:
`npx playwright install chromium`. A missing reference or browser fails it with
the reason; it never skips.

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

## The visual loop

`tests/visual/states.json` is the ruler: which states are measured, how, and the
limit with the reasoning that chose it. `docs/loop/done.toml` protects it and the
reference images — a change there changes the measurement and has to be made
deliberately, never to make a run pass. When new references arrive, measure the
new design's own floor (its mock rendered by the same Chromium) before trusting
the old limit.
