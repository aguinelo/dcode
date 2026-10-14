// Names the development Electron "DCode Box" on macOS, before `npm start`.
//
// Run from the sources, the app is Electron's own bundle, and macOS reads the
// name in the menu bar and under the Dock icon from that bundle's Info.plist —
// not from app.setName. A packaged app carries its own bundle and needs none of
// this. The bundle is in node_modules, so `npm ci` brings Electron's name back
// and this runs again on the next start; it does nothing when the name is
// already there.
//
// Editing Info.plist breaks the bundle's signature, so the bundle is signed
// again ad hoc, the way Electron's own development build is signed.
//
// A failure never stops the app from starting: it is said, and the menu bar
// keeps saying "Electron".

import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

export const DEV_NAME = 'DCode Box';

/** The .app folder an executable inside `X.app/Contents/MacOS/` belongs to, or null. */
export function bundleRoot(executable) {
  const macos = path.dirname(executable);
  const contents = path.dirname(macos);
  const app = path.dirname(contents);
  if (path.basename(macos) !== 'MacOS' || path.basename(contents) !== 'Contents' || !app.endsWith('.app')) return null;
  return app;
}

function read(plist, key) {
  try {
    return execFileSync('/usr/libexec/PlistBuddy', ['-c', `Print :${key}`, plist], { encoding: 'utf8' }).trim();
  } catch {
    return null;
  }
}

function setKey(plist, key, value) {
  execFileSync('/usr/bin/plutil', ['-replace', key, '-string', value, plist]);
}

export function nameDevBundle() {
  if (process.platform !== 'darwin') return;
  // Requiring electron downloads its binary when it is missing, and answers its path.
  const executable = createRequire(import.meta.url)('electron');
  const app = bundleRoot(executable);
  if (!app) {
    console.warn(`dev-name: ${executable} is not inside a macOS .app; the menu bar keeps Electron's name.`);
    return;
  }
  const plist = path.join(app, 'Contents', 'Info.plist');
  if (!existsSync(plist)) {
    console.warn(`dev-name: ${plist} does not exist; the menu bar keeps Electron's name.`);
    return;
  }
  if (read(plist, 'CFBundleName') === DEV_NAME && read(plist, 'CFBundleDisplayName') === DEV_NAME) return;
  setKey(plist, 'CFBundleName', DEV_NAME);
  setKey(plist, 'CFBundleDisplayName', DEV_NAME);
  execFileSync('/usr/bin/codesign', ['--force', '--deep', '--sign', '-', app]);
  // LaunchServices caches a bundle's name; touching the bundle makes it read it again.
  execFileSync('/usr/bin/touch', [app]);
  console.log(`dev-name: the development Electron is named ${DEV_NAME}.`);
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? '').href) {
  try {
    nameDevBundle();
  } catch (err) {
    console.warn(`dev-name: could not name the development Electron, so the menu bar keeps saying "Electron": ${err instanceof Error ? err.message : String(err)}`);
  }
}
