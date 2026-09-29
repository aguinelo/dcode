// The person's own arrangement of the sidebar: project order, labels and which
// projects are collapsed. Local to this client by design — the daemon never
// learns them, and renaming a project changes its label, never its directory.

export interface Prefs {
  /** Project ids (workspace paths) in the order the person put them. */
  order: string[];
  labels: Record<string, string>;
  collapsed: Record<string, boolean>;
}

export const emptyPrefs: Prefs = { order: [], labels: {}, collapsed: {} };

export const PREFS_KEY = 'dcode.desktop.prefs.v1';

function isStringRecord<T extends 'string' | 'boolean'>(v: unknown, kind: T): v is Record<string, T extends 'string' ? string : boolean> {
  return (
    typeof v === 'object' &&
    v !== null &&
    !Array.isArray(v) &&
    Object.values(v as Record<string, unknown>).every((x) => typeof x === kind)
  );
}

/**
 * Reads what was stored. Nothing stored is the ordinary first run; something
 * stored that cannot be read is a problem, returned so it can be said — the
 * defaults are used either way, but never in silence.
 */
export function parsePrefs(raw: string | null): { prefs: Prefs | null; problem: string | null } {
  if (raw === null) return { prefs: null, problem: null };
  let data: unknown;
  try {
    data = JSON.parse(raw);
  } catch (err) {
    return { prefs: null, problem: `preferências locais ilegíveis (${(err as Error).message})` };
  }
  if (typeof data !== 'object' || data === null) {
    return { prefs: null, problem: 'preferências locais ilegíveis (não são um objeto)' };
  }
  const d = data as Record<string, unknown>;
  const order = Array.isArray(d.order) && d.order.every((x) => typeof x === 'string') ? (d.order as string[]) : null;
  const labels = isStringRecord(d.labels, 'string') ? d.labels : null;
  const collapsed = isStringRecord(d.collapsed, 'boolean') ? d.collapsed : null;
  if (!order || !labels || !collapsed) {
    return { prefs: null, problem: 'preferências locais num formato que esta versão não lê' };
  }
  return { prefs: { order, labels, collapsed }, problem: null };
}

export function setCollapsed(p: Prefs, id: string, collapsed: boolean): Prefs {
  return { ...p, collapsed: { ...p.collapsed, [id]: collapsed } };
}

export function setAllCollapsed(p: Prefs, ids: readonly string[], collapsed: boolean): Prefs {
  const next = { ...p.collapsed };
  for (const id of ids) next[id] = collapsed;
  return { ...p, collapsed: next };
}

/** An empty label keeps the one before it: clearing a name is not a way to lose it. */
export function relabel(p: Prefs, id: string, label: string): Prefs {
  const trimmed = label.trim();
  if (!trimmed) return p;
  return { ...p, labels: { ...p.labels, [id]: trimmed } };
}

/**
 * Moves a project to a position among the ones on screen. `shown` is the order
 * the sidebar is drawing now, which becomes the stored order — so a move is
 * always relative to what the person sees.
 */
export function moveProject(p: Prefs, shown: readonly string[], id: string, to: number): Prefs {
  const rest = shown.filter((x) => x !== id);
  const at = Math.max(0, Math.min(to, rest.length));
  rest.splice(at, 0, id);
  return { ...p, order: rest };
}

/** Drops `id` just before `target`, the way dragging a row onto another does. */
export function dropBefore(p: Prefs, shown: readonly string[], id: string, target: string): Prefs {
  if (id === target) return p;
  const rest = shown.filter((x) => x !== id);
  const at = rest.indexOf(target);
  if (at < 0) return p;
  rest.splice(at, 0, id);
  return { ...p, order: rest };
}
