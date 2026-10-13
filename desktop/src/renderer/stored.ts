import { emptyPrefs, parsePrefs, PREFS_KEY, type Prefs } from '../state/prefs';
import type { Host } from './host';
import type { Look } from './TopBar';

// What the window keeps in its own storage and reads before its first frame:
// the version of the screen and the sidebar's arrangement. A storage that
// cannot be read is said, never a silent default.

/** Which version of the screen, per window, as a convenience: the grid when unknown (D32). */
export const LOOK_KEY = 'dcode.desktop.look.v1';

export function loadLook(): Look {
  try {
    return window.localStorage.getItem(LOOK_KEY) === 'crew' ? 'crew' : 'grade';
  } catch {
    return 'grade';
  }
}

export function loadPrefs(host: Host): { prefs: Prefs; problem: string | null } {
  let raw: string | null;
  try {
    raw = window.localStorage.getItem(PREFS_KEY);
  } catch (err) {
    return { prefs: host.initialPrefs, problem: `Não foi possível ler as preferências locais (${(err as Error).message}); a lateral começa do arranjo inicial.` };
  }
  const read = parsePrefs(raw);
  if (read.problem) return { prefs: host.initialPrefs, problem: `${read.problem}; a lateral começa do arranjo inicial.` };
  return { prefs: read.prefs ?? host.initialPrefs ?? emptyPrefs, problem: null };
}
