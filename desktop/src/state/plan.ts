// The plan of the turn under way, pinned at the top of the conversation (D57).
//
// plan.updated is folded into one block where it first appeared (session.ts);
// while the turn that last revised it runs, that block is read here and drawn
// above the flow, and the flow leaves it out. When the turn ends the pin goes
// and the block is back where it was. Pure: statuses are the event's own.

import type * as P from '../protocol/generated';
import type { Entry, SessionView } from './session';

export interface PinnedPlan {
  items: P.PlanItem[];
  /** The step under way, counted from 1. */
  step: number;
  total: number;
  /** What that step says. */
  current: string;
}

/**
 * The step under way: the active one; with none active, the first not yet
 * done; with every step done, the last.
 */
function stepOf(items: P.PlanItem[]): number {
  const active = items.findIndex((i) => i.status === 'active');
  if (active >= 0) return active;
  const open = items.findIndex((i) => i.status !== 'done');
  return open >= 0 ? open : items.length - 1;
}

/** The plan to pin, or null: no turn under way, no plan it revised, or an empty one. */
export function pinnedPlan(view: Pick<SessionView, 'turn' | 'entries'>): PinnedPlan | null {
  if (!view.turn) return null;
  const plan = view.entries.find((e) => e.kind === 'plan');
  if (!plan || plan.kind !== 'plan' || plan.turnId !== view.turn.id || plan.items.length === 0) return null;
  const at = stepOf(plan.items);
  return { items: plan.items, step: at + 1, total: plan.items.length, current: plan.items[at]?.text ?? '' };
}

/** The flow's entries without the plan block while it is pinned: drawn once, not twice. */
export function withoutPinned(entries: Entry[], pin: PinnedPlan | null): Entry[] {
  return pin ? entries.filter((e) => e.kind !== 'plan') : entries;
}
