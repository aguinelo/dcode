// The grid: the conversations that want the person's attention, laid out at
// once. Pure — what the person keeps and has seen comes in as Attention, and
// nothing here reads a clock or storage.
//
// It fills itself: what waits for you, what you pinned, what is running, and
// what finished since you last looked. Places are stable — a conversation keeps
// its place while it stays, a new one takes the next, and the grid closes the
// gap one leaves — so panels do not move under the eye while the work moves.

import type { Attention } from './attention';
import type { Row } from './sidebar';

/** Why a conversation is on the grid, in the order they win a place. */
export type Claim = 'waiting' | 'pinned' | 'running' | 'unseen';

const RANK: Readonly<Record<Claim, number>> = { waiting: 0, pinned: 1, running: 2, unseen: 3 };

function instant(s: string | null | undefined): number {
  const t = Date.parse(s ?? '');
  return Number.isNaN(t) ? Number.NaN : t;
}

/**
 * A conversation whose last turn ended after the person last looked at it —
 * and after this window first ran, so an old record is not news.
 */
export function finishedUnseen(r: Row, a: Attention): boolean {
  if (r.state === 'running' || r.state === 'blocked') return false;
  if (r.lastEvent !== 'turn.completed') return false;
  const at = instant(r.lastAt);
  if (Number.isNaN(at)) return false;
  const since = instant(a.since);
  if (!Number.isNaN(since) && at <= since) return false;
  const seen = instant(a.seen[r.id]);
  return Number.isNaN(seen) || at > seen;
}

export function claimOf(r: Row, a: Attention): Claim | null {
  if (r.state === 'blocked') return 'waiting';
  if (a.pinned.includes(r.id)) return 'pinned';
  if (r.state === 'running') return 'running';
  if (finishedUnseen(r, a)) return 'unseen';
  return null;
}

export interface GridPlan {
  /** The conversations on the grid, in their places. */
  ids: string[];
  /** Why each is there. */
  claims: Record<string, Claim>;
  /** What wanted a place and found none, best first. */
  overflow: Row[];
}

/**
 * Lays out the grid. `previous` is the grid as drawn now: whoever stays keeps
 * the order they had, and newcomers follow in the order they win a place.
 * When more want a place than there are, what waits for you wins, then what
 * you pinned, then what runs, then what finished — the newest first in each.
 */
export function planGrid(rows: readonly Row[], a: Attention, previous: readonly string[], capacity: number): GridPlan {
  const claims: Record<string, Claim> = {};
  const candidates: Row[] = [];
  for (const r of rows) {
    const c = claimOf(r, a);
    if (c) {
      claims[r.id] = c;
      candidates.push(r);
    }
  }
  const latest = (r: Row) => {
    const t = instant(r.lastAt);
    return Number.isNaN(t) ? 0 : t;
  };
  candidates.sort((x, y) => RANK[claims[x.id] as Claim] - RANK[claims[y.id] as Claim] || latest(y) - latest(x));
  const chosen = new Set(candidates.slice(0, Math.max(0, capacity)).map((r) => r.id));
  const kept = previous.filter((id) => chosen.has(id));
  const added = candidates.filter((r) => chosen.has(r.id) && !kept.includes(r.id)).map((r) => r.id);
  return { ids: [...kept, ...added], claims, overflow: candidates.filter((r) => !chosen.has(r.id)) };
}

/** Columns and rows for `n` panels: one fills the stage, two sit side by side, then 2×2, 3×2, 3×3. */
export function gridShape(n: number): { cols: number; rows: number } {
  if (n <= 1) return { cols: 1, rows: 1 };
  if (n === 2) return { cols: 2, rows: 1 };
  if (n <= 4) return { cols: 2, rows: 2 };
  if (n <= 6) return { cols: 3, rows: 2 };
  return { cols: 3, rows: 3 };
}

/** How many panels fit: six, nine on a large screen. */
export function capacityFor(width: number, height: number): number {
  return width >= 1700 && height >= 1000 ? 9 : 6;
}
