// `npm run gen:icons`: every size of the app icon from assets/icon/icon.svg
// (D49), so the drawing is the one source and the sizes are never edited by
// hand. Rasterised by Playwright's Chromium, the same one check:visual uses;
// the .icns and the .ico are packed here, with no platform tool, so the set
// is the same whichever machine makes it.
//
//   icon.icns  macOS, PNG entries 16 to 1024 (packagerConfig.icon)
//   icon.ico   Windows, PNG entries 16 to 256 (packagerConfig.icon)
//   icon.png   Linux, 512
//
// A missing browser or source fails with the reason; nothing is skipped.

import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const here = path.dirname(fileURLToPath(import.meta.url));
const dir = path.resolve(here, '..', 'assets', 'icon');

async function rasterize(svg, sizes) {
  let browser;
  try {
    browser = await chromium.launch();
  } catch (err) {
    throw new Error(`Playwright's Chromium did not start (${err.message.split('\n')[0]}); install it with: npx playwright install chromium`, {
      cause: err,
    });
  }
  const src = `data:image/svg+xml;base64,${Buffer.from(svg).toString('base64')}`;
  const out = new Map();
  try {
    for (const size of sizes) {
      const page = await browser.newPage({ viewport: { width: size, height: size }, deviceScaleFactor: 1 });
      // setContent waits for the load event, which waits for the image.
      await page.setContent(`<html><body style="margin:0;background:transparent"><img src="${src}" width="${size}" height="${size}" style="display:block"></body></html>`);
      const drawn = await page.locator('img').evaluate((img) => img.naturalWidth > 0);
      if (!drawn) throw new Error('Chromium did not draw icon.svg');
      out.set(size, await page.screenshot({ omitBackground: true, clip: { x: 0, y: 0, width: size, height: size } }));
      await page.close();
    }
  } finally {
    await browser.close();
  }
  return out;
}

function u32(n) {
  const b = Buffer.alloc(4);
  b.writeUInt32BE(n);
  return b;
}

// Apple's icon family: a header and one chunk per size, each a whole PNG.
function icns(png) {
  const entries = [
    ['icp4', 16],
    ['icp5', 32],
    ['icp6', 64],
    ['ic07', 128],
    ['ic08', 256],
    ['ic09', 512],
    ['ic10', 1024],
    ['ic11', 32],
    ['ic12', 64],
    ['ic13', 256],
    ['ic14', 512],
  ];
  const chunks = entries.map(([type, size]) => Buffer.concat([Buffer.from(type, 'ascii'), u32(8 + png.get(size).length), png.get(size)]));
  const body = Buffer.concat(chunks);
  return Buffer.concat([Buffer.from('icns', 'ascii'), u32(8 + body.length), body]);
}

// The ICO directory with PNG images, which Windows reads from Vista on.
function ico(png, sizes) {
  const header = Buffer.alloc(6);
  header.writeUInt16LE(0, 0);
  header.writeUInt16LE(1, 2);
  header.writeUInt16LE(sizes.length, 4);
  let offset = 6 + 16 * sizes.length;
  const dirs = [];
  for (const size of sizes) {
    const data = png.get(size);
    const d = Buffer.alloc(16);
    d.writeUInt8(size >= 256 ? 0 : size, 0);
    d.writeUInt8(size >= 256 ? 0 : size, 1);
    d.writeUInt16LE(1, 4);
    d.writeUInt16LE(32, 6);
    d.writeUInt32LE(data.length, 8);
    d.writeUInt32LE(offset, 12);
    offset += data.length;
    dirs.push(d);
  }
  return Buffer.concat([header, ...dirs, ...sizes.map((s) => png.get(s))]);
}

async function main() {
  const svg = await readFile(path.join(dir, 'icon.svg'), 'utf8');
  const icoSizes = [16, 24, 32, 48, 64, 128, 256];
  const png = await rasterize(svg, [16, 24, 32, 48, 64, 128, 256, 512, 1024]);
  await writeFile(path.join(dir, 'icon.icns'), icns(png));
  await writeFile(path.join(dir, 'icon.ico'), ico(png, icoSizes));
  await writeFile(path.join(dir, 'icon.png'), png.get(512));
  console.log(`wrote ${path.relative(process.cwd(), dir)}/icon.{icns,ico,png}`);
}

main().catch((err) => {
  console.error(`gen:icons failed: ${err.message}`);
  process.exit(1);
});
