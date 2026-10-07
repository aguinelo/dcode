// What the grid remembers of the person, local to this window: what they keep
// on it, and when they last looked at each conversation. The daemon never
// learns it, as it never learns the sidebar's arrangement (D7).

export interface Attention {
  /** Conversation ids kept on the grid whatever their state. */
  pinned: string[];
  /** For each conversation looked at: its last activity at that moment. */
  seen: Record<string, string>;
  /**
   * When this window first ran. What finished before it reads as seen, or
   * the first run would fill the grid with every conversation ever recorded.
   */
  since: string | null;
}

export const emptyAttention: Attention = { pinned: [], seen: {}, since: null };

export const ATTENTION_KEY = 'dcode.desktop.attention.v1';

function isStringRecord(v: unknown): v is Record<string, string> {
  return typeof v === 'object' && v !== null && !Array.isArray(v) && Object.values(v).every((x) => typeof x === 'string');
}

/**
 * Reads what was stored. Nothing stored is the ordinary first run; something
 * stored that cannot be read is a problem, returned so it can be said.
 */
export function parseAttention(raw: string | null): { attention: Attention | null; problem: string | null } {
  if (raw === null) return { attention: null, problem: null };
  let data: unknown;
  try {
    data = JSON.parse(raw);
  } catch (err) {
    return { attention: null, problem: `memória da grade ilegível (${(err as Error).message})` };
  }
  const d = (typeof data === 'object' && data !== null ? data : {}) as Record<string, unknown>;
  const pinned = Array.isArray(d.pinned) && d.pinned.every((x) => typeof x === 'string') ? (d.pinned as string[]) : null;
  const seen = isStringRecord(d.seen) ? d.seen : null;
  const since = typeof d.since === 'string' || d.since === null ? (d.since as string | null) : undefined;
  if (!pinned || !seen || since === undefined) return { attention: null, problem: 'memória da grade num formato que esta versão não lê' };
  return { attention: { pinned, seen, since }, problem: null };
}

/** Marks when this window first ran, once. */
export function beginAt(a: Attention, now: string): Attention {
  return a.since === null ? { ...a, since: now } : a;
}

export function togglePin(a: Attention, id: string): Attention {
  return a.pinned.includes(id) ? { ...a, pinned: a.pinned.filter((x) => x !== id) } : { ...a, pinned: [...a.pinned, id] };
}

/** The person looked at a conversation as it stood at `lastAt`: what came before is seen. */
export function markSeen(a: Attention, id: string, lastAt: string | null): Attention {
  if (!lastAt || a.seen[id] === lastAt) return a;
  return { ...a, seen: { ...a.seen, [id]: lastAt } };
}

/**
 * Forgets conversations that no longer exist, so the memory does not grow
 * with every conversation the daemon has pruned since.
 */
export function forgetMissing(a: Attention, known: ReadonlySet<string>): Attention {
  const pinned = a.pinned.filter((id) => known.has(id));
  const seenIds = Object.keys(a.seen);
  const kept = seenIds.filter((id) => known.has(id));
  if (pinned.length === a.pinned.length && kept.length === seenIds.length) return a;
  const seen: Record<string, string> = {};
  for (const id of kept) seen[id] = a.seen[id] as string;
  return { ...a, pinned, seen };
}
