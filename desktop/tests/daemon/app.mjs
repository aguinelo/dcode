// The window under test: the built app, launched as Electron by Playwright,
// driven the way a person drives it — clicks and keys — and read through the
// attributes listed in docs/loop/tasks.md, "O que a régua lê".

import electronBinary from 'electron';
import { _electron as electron } from 'playwright';

/**
 * Launches the built app (`npm run build` into .vite/) with its own user data,
 * so nothing stored by a real run — a collapsed project, an order — can hide a
 * row the check looks for.
 */
export async function launchApp({ desktop, env, userData }) {
  const app = await electron.launch({
    executablePath: electronBinary,
    args: [desktop],
    cwd: desktop,
    env: { ...process.env, ...env, DCODE_DESKTOP_USER_DATA: userData },
    timeout: 30_000,
  });
  const logs = [];
  app.process().stdout?.on('data', (c) => logs.push(`main: ${String(c).trimEnd()}`));
  app.process().stderr?.on('data', (c) => logs.push(`main: ${String(c).trimEnd()}`));
  const page = await app.firstWindow({ timeout: 30_000 });
  page.on('console', (m) => {
    if (m.type() === 'error' || m.type() === 'warning') logs.push(`renderer ${m.type()}: ${m.text()}`);
  });
  page.on('pageerror', (e) => logs.push(`renderer page error: ${e.message}`));
  await page.waitForFunction(() => ['true', 'error'].includes(document.documentElement.dataset.ready ?? ''), null, { timeout: 20_000 });
  const ready = await page.evaluate(() => [document.documentElement.dataset.ready, document.documentElement.dataset.error ?? '']);
  if (ready[0] !== 'true') throw new Error(`the window did not become ready: ${ready[1]}`);

  let closed = false;
  return {
    app,
    page,
    logs,
    /** Quits the way the menu does (app.quit), and fails if it does not end. */
    async close() {
      if (closed) return;
      closed = true;
      const quit = app.close();
      const late = new Promise((_, reject) => setTimeout(() => reject(new Error('the app did not quit within 10 s')), 10_000));
      try {
        await Promise.race([quit, late]);
      } catch (err) {
        app.process().kill('SIGKILL');
        throw err;
      }
    },
    /** For cleanup after a failure: no questions, no waiting. */
    kill() {
      closed = true;
      app.process().kill('SIGKILL');
    },
  };
}

/**
 * Answers the folder picker with `dir`, in the main process. The app must ask
 * through `dialog.showOpenDialog` on the `dialog` object, which is what is
 * replaced here — a native sheet no test can click.
 */
export async function answerFolderPicker(app, dir) {
  await app.evaluate(({ dialog }, chosen) => {
    dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [chosen] });
  }, dir);
}

/** Waits for a locator to be visible, failing with what was being waited for. */
export async function visible(locator, timeoutMs, what) {
  try {
    await locator.first().waitFor({ state: 'visible', timeout: timeoutMs });
  } catch {
    throw new Error(`${what} did not appear within ${timeoutMs} ms`);
  }
  return locator.first();
}

/** Waits for the daemon status to read `state`, saying what it read instead. */
export async function daemonState(page, state, timeoutMs) {
  const want = page.locator(`[data-daemon="${state}"]`);
  try {
    await want.first().waitFor({ state: 'visible', timeout: timeoutMs });
  } catch {
    const seen = await page
      .locator('[data-daemon]')
      .evaluateAll((els) => els.map((e) => `${e.getAttribute('data-daemon')} (“${(e.textContent ?? '').trim()}”)`));
    throw new Error(`expected [data-daemon="${state}"] within ${timeoutMs} ms; the window has ${seen.join(', ') || 'no [data-daemon] element'}`);
  }
  return want.first();
}

/** Polls `probe` until it returns something truthy, or fails with `what`. */
export async function eventually(probe, timeoutMs, what) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const v = await probe();
    if (v) return v;
    if (Date.now() > deadline) throw new Error(`${what} (waited ${timeoutMs} ms)`);
    await new Promise((r) => setTimeout(r, 150));
  }
}
