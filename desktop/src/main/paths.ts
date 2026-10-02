import path from 'node:path';

/** The folder under the system's data folder that is the app's alone. */
export const USER_DATA_NAME = 'dcode-desktop';

/**
 * Where Chromium keeps the window's own data: the preferences in
 * localStorage, caches, cookies.
 *
 * Never Electron's default. That is the product name inside the system's data
 * folder — on macOS `~/Library/Application Support/DCode` — and the file
 * system there does not tell DCode from dcode, the folder dcode itself keeps
 * its configuration, records and state in. Deleting the app's data must not
 * take dcode's configuration with it.
 *
 * DCODE_DESKTOP_USER_DATA names another folder, which must be absolute: the
 * check gives each scenario its own.
 */
export function userDataDir(
  env: Readonly<Record<string, string | undefined>>,
  appData: string,
): { ok: true; dir: string } | { ok: false; reason: string } {
  const chosen = env.DCODE_DESKTOP_USER_DATA;
  if (chosen) {
    if (!path.isAbsolute(chosen)) {
      return { ok: false, reason: `DCODE_DESKTOP_USER_DATA precisa ser um caminho absoluto, e é “${chosen}”.` };
    }
    return { ok: true, dir: chosen };
  }
  return { ok: true, dir: path.join(appData, USER_DATA_NAME) };
}
