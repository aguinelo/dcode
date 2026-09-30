// `npm run check:daemon`: does the window talk to a real daemon?
//
// Builds the app and `dcode` from this checkout, then runs each scenario in
// scenarios.mjs against a fresh `dcode serve`, a fresh scripted model and a
// fresh copy of the window, launched as Electron. Prints one line per
// scenario, writes report.json — with the window's logs and the daemon's
// output — and a screenshot of each failure to test-results/daemon/, and exits
// non-zero when any scenario fails, or when anything it needs is missing:
// that is a failure, never a skip.
//
// `npm run check:daemon -- <name> …` runs only the scenarios named.

import { spawnSync } from 'node:child_process';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { launchApp } from './app.mjs';
import { daemonEnv, dcodeBinary, startDaemon, workspace } from './daemon.mjs';
import { startModel } from './model.mjs';
import { scenarios } from './scenarios.mjs';
import { follow } from './wire.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, '..', '..');
const repo = path.resolve(desktop, '..');
const out = path.join(desktop, 'test-results', 'daemon');

function buildApp() {
  const r = spawnSync(process.execPath, ['scripts/build.mjs'], { cwd: desktop, encoding: 'utf8' });
  if (r.status !== 0) throw new Error(`npm run build failed:\n${r.stdout}${r.stderr}`);
}

/** Everything one scenario opens, closed in reverse whatever happened. */
function context(bin) {
  const cleanups = [];
  const state = { ui: null, daemons: [], models: [] };
  const ctx = {
    async fresh() {
      // Short, because a Unix socket path is capped near 104 bytes on macOS.
      const dir = await mkdtemp(path.join(os.tmpdir(), 'dck-'));
      cleanups.push(() => rm(dir, { recursive: true, force: true }));
      return {
        dir,
        socket: path.join(dir, 'd.sock'),
        home: path.join(dir, 'home'),
        userData: path.join(dir, 'user-data'),
        ws: await workspace(path.join(dir, 'ws')),
      };
    },
    env(t, model) {
      return { ...daemonEnv(t.home, model.url, t.socket), DCODE_BIN: bin };
    },
    async model() {
      const m = await startModel();
      state.models.push(m);
      cleanups.push(() => m.close());
      return m;
    },
    async daemon(env, cwd) {
      const d = await startDaemon(bin, env, cwd);
      state.daemons.push(d);
      cleanups.push(() => d.stop());
      return d;
    },
    follow(socket, id) {
      const f = follow(socket, id);
      cleanups.push(() => f.close());
      return f;
    },
    async launch(env, t) {
      const ui = await launchApp({ desktop, env, userData: t.userData });
      state.ui = ui;
      cleanups.push(() => ui.kill());
      return ui;
    },
  };
  return {
    ctx,
    state,
    async cleanup() {
      for (const c of cleanups.reverse()) {
        try {
          await c();
        } catch {
          // Cleanup of something already gone; the scenario's own verdict stands.
        }
      }
    },
  };
}

async function runOne(scenario, bin) {
  const { ctx, state, cleanup } = context(bin);
  const started = Date.now();
  let error = null;
  try {
    await scenario.run(ctx);
  } catch (err) {
    error = err instanceof Error ? err.message : String(err);
    if (state.ui) {
      await state.ui.page.screenshot({ path: path.join(out, `${scenario.name}.png`) }).catch(() => {});
    }
  }
  const unexpected = state.models.flatMap((m) => m.unexpected);
  if (!error && unexpected.length > 0) error = `the model was asked what the scenario did not script: ${unexpected.join('; ')}`;
  const result = {
    name: scenario.name,
    about: scenario.about,
    ok: error === null,
    error,
    ms: Date.now() - started,
    window: state.ui?.logs ?? [],
    daemon: state.daemons.flatMap((d) => d.log),
  };
  await cleanup();
  return result;
}

async function main() {
  if (process.platform === 'linux' && !process.env.DISPLAY && !process.env.WAYLAND_DISPLAY) {
    throw new Error('Electron needs a display: on Linux run this under xvfb-run');
  }
  const only = process.argv.slice(2).filter((a) => !a.startsWith('-'));
  const unknown = only.filter((n) => !scenarios.some((s) => s.name === n));
  if (unknown.length > 0) throw new Error(`no scenario named ${unknown.join(', ')}; there are ${scenarios.map((s) => s.name).join(', ')}`);
  const chosen = only.length > 0 ? scenarios.filter((s) => only.includes(s.name)) : scenarios;

  await rm(out, { recursive: true, force: true });
  await mkdir(out, { recursive: true });
  buildApp();
  const bin = dcodeBinary(repo, out);

  const results = [];
  for (const scenario of chosen) {
    const r = await runOne(scenario, bin);
    results.push(r);
    console.log(`${r.ok ? 'ok  ' : 'FAIL'} ${r.name.padEnd(16)} ${r.ok ? scenario.about : r.error}`);
  }
  await writeFile(path.join(out, 'report.json'), JSON.stringify({ bin, results }, null, 2));
  const failed = results.filter((r) => !r.ok);
  console.log(`${results.length - failed.length} of ${results.length} scenarios pass; report: ${path.relative(desktop, path.join(out, 'report.json'))}`);
  if (failed.length > 0) process.exit(1);
}

main().catch((err) => {
  console.error(`check:daemon could not run: ${err?.stack ?? err}`);
  process.exit(1);
});
