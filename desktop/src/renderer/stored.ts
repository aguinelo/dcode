import { emptyPrefs, parsePrefs, PREFS_KEY, type Prefs } from '../state/prefs';
import type { Host } from './host';
import type { Look } from './TopBar';

// What the window keeps in its own storage and reads before its first frame:
// the version of the screen and the sidebar's arrangement. A storage that
// cannot be read is said, never a silent default.

/** Which version of the screen, per window, as a convenience (D32): the Crew when unknown (D63). */
export const LOOK_KEY = 'dcode.desktop.look.v1';

/** The Crew is the main screen (D63); the grid is the alternative, kept when chosen. */
export const DEFAULT_LOOK: Look = 'crew';

/** What a stored value comes to: either version as chosen, else the fallback. */
export function lookOf(raw: string | null, fallback: Look = DEFAULT_LOOK): Look {
  return raw === 'crew' || raw === 'grade' ? raw : fallback;
}

export function loadLook(fallback: Look = DEFAULT_LOOK): Look {
  try {
    return lookOf(window.localStorage.getItem(LOOK_KEY), fallback);
  } catch {
    return fallback;
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
