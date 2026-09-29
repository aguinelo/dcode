// `npm run check:visual`: how far the window is from the design.
//
// Builds the renderer, opens it in fixture mode in Playwright's Chromium, one
// page per reference state, and compares each screenshot with the reference
// PNG of the same name using pixelmatch. Writes the screenshot, the diff and a
// report.json to test-results/visual/, prints one line per state, and exits
// non-zero when any state is past the limit in states.json — or when anything
// needed to measure is missing, which is a failure and never a skip.

import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import pixelmatch from 'pixelmatch';
import { chromium } from 'playwright';
import { PNG } from 'pngjs';
import { build } from 'vite';

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, '..', '..');
const repo = path.resolve(desktop, '..');
const out = path.join(desktop, 'test-results', 'visual');
const ORIGIN = 'http://dcode.fixture';

const TYPES = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.woff2': 'font/woff2',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
};

function referenceDir(config) {
  const flag = process.argv.indexOf('--refs');
  if (flag >= 0) {
    const dir = process.argv[flag + 1];
    if (!dir) throw new Error('--refs needs a directory');
    return path.resolve(dir);
  }
  if (process.env.DCODE_DESIGN_SCREENSHOTS) return path.resolve(process.env.DCODE_DESIGN_SCREENSHOTS);
  return path.resolve(repo, config.reference.dir);
}

async function readReference(dir, state, crop) {
  const file = path.join(dir, `${state}.png`);
  let raw;
  try {
    raw = await readFile(file);
  } catch (err) {
    throw new Error(`no reference for ${state} at ${file} (${err.code ?? err.message}); pass --refs <dir> or DCODE_DESIGN_SCREENSHOTS`, {
      cause: err,
    });
  }
  const png = PNG.sync.read(raw);
  if (png.width < crop.x + crop.width || png.height < crop.y + crop.height) {
    throw new Error(`${file} is ${png.width}×${png.height}, smaller than the crop in states.json`);
  }
  const cropped = new PNG({ width: crop.width, height: crop.height });
  PNG.bitblt(png, cropped, crop.x, crop.y, crop.width, crop.height, 0, 0);
  return cropped;
}

async function buildRenderer(dir) {
  await build({
    configFile: path.join(desktop, 'vite.renderer.config.ts'),
    root: desktop,
    logLevel: 'warn',
    build: { outDir: dir, emptyOutDir: true },
  });
}

async function serve(context, dir) {
  const blocked = [];
  await context.route('**/*', async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== ORIGIN) {
      // The window loads nothing from the network; a request out is a defect.
      blocked.push(url.href);
      return route.abort();
    }
    const file = path.join(dir, decodeURIComponent(url.pathname));
    if (!file.startsWith(dir)) return route.abort();
    try {
      const body = await readFile(file);
      return route.fulfill({ status: 200, body, contentType: TYPES[path.extname(file)] ?? 'application/octet-stream' });
    } catch {
      return route.fulfill({ status: 404, body: `not found: ${url.pathname}` });
    }
  });
  return blocked;
}

async function capture(context, state, config) {
  const page = await context.newPage();
  const problems = [];
  page.on('console', (m) => {
    if (m.type() === 'error') problems.push(`console: ${m.text()}`);
  });
  page.on('pageerror', (e) => problems.push(`page error: ${e.message}`));
  await page.clock.setFixedTime(new Date(config.clock));
  await page.goto(`${ORIGIN}/index.html?fixture=${encodeURIComponent(state)}`);
  await page.waitForFunction(() => ['true', 'error'].includes(document.documentElement.dataset.ready ?? ''), null, {
    timeout: 20_000,
  });
  const ready = await page.evaluate(() => [document.documentElement.dataset.ready, document.documentElement.dataset.error ?? '', document.body.innerText.slice(0, 300)]);
  if (ready[0] !== 'true') problems.push(`the window did not become ready: ${ready[1] || ready[2]}`);
  const shot = PNG.sync.read(await page.screenshot({ animations: 'disabled', caret: 'hide' }));
  await page.close();
  return { shot, problems };
}

async function main() {
  const config = JSON.parse(await readFile(path.join(here, 'states.json'), 'utf8'));
  const refs = referenceDir(config);
  await rm(out, { recursive: true, force: true });
  await mkdir(out, { recursive: true });
  const rendered = path.join(out, 'renderer');
  await buildRenderer(rendered);

  let browser;
  try {
    browser = await chromium.launch();
  } catch (err) {
    throw new Error(`Playwright's Chromium did not start (${err.message.split('\n')[0]}); install it with: npx playwright install chromium`, {
      cause: err,
    });
  }
  const { width, height } = config.viewport;
  const context = await browser.newContext({
    viewport: { width, height },
    deviceScaleFactor: 1,
    colorScheme: config.colorScheme,
    reducedMotion: config.reducedMotion,
  });
  const blocked = await serve(context, rendered);

  const report = { generatedAt: new Date().toISOString(), references: refs, limit: config.maxMismatchRatio, states: {} };
  let failed = false;
  for (const state of config.states) {
    const entry = { pass: false };
    report.states[state] = entry;
    try {
      const reference = await readReference(refs, state, config.reference.crop);
      const { shot, problems } = await capture(context, state, config);
      if (shot.width !== reference.width || shot.height !== reference.height) {
        throw new Error(`screenshot is ${shot.width}×${shot.height}, reference crop is ${reference.width}×${reference.height}`);
      }
      const diff = new PNG({ width, height });
      const mismatched = pixelmatch(reference.data, shot.data, diff.data, width, height, config.pixelmatch);
      await writeFile(path.join(out, `${state}.png`), PNG.sync.write(shot));
      await writeFile(path.join(out, `${state}.diff.png`), PNG.sync.write(diff));
      await writeFile(path.join(out, `${state}.reference.png`), PNG.sync.write(reference));
      Object.assign(entry, {
        mismatchedPixels: mismatched,
        totalPixels: width * height,
        ratio: mismatched / (width * height),
        problems,
        actual: `${state}.png`,
        diff: `${state}.diff.png`,
      });
      entry.pass = entry.ratio <= config.maxMismatchRatio && problems.length === 0;
      const pct = (entry.ratio * 100).toFixed(2);
      console.log(
        `${entry.pass ? 'ok  ' : 'FAIL'} ${state}  mismatch ${pct}% (${mismatched} px)  limit ${(config.maxMismatchRatio * 100).toFixed(2)}%${problems.length ? `  ${problems.join('; ')}` : ''}`,
      );
    } catch (err) {
      entry.error = err.message;
      console.log(`FAIL ${state}  ${err.message}`);
    }
    failed ||= !entry.pass;
  }
  await browser.close();
  if (blocked.length > 0) {
    report.networkRequests = blocked;
    console.log(`FAIL the window asked the network for: ${blocked.join(', ')}`);
    failed = true;
  }
  report.pass = !failed;
  await writeFile(path.join(out, 'report.json'), `${JSON.stringify(report, null, 2)}\n`);
  console.log(`report: ${path.relative(desktop, path.join(out, 'report.json'))}`);
  process.exit(failed ? 1 : 0);
}

main().catch((err) => {
  console.error(`check:visual could not measure: ${err.message}`);
  process.exit(1);
});
