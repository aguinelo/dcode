// The plan of the turn under way, pinned at the top of the conversation (D57).

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

export function pinnedPlan(_view: Pick<SessionView, 'turn' | 'entries'>): PinnedPlan | null {
  return null;
}

export function withoutPinned(entries: Entry[], _pin: PinnedPlan | null): Entry[] {
  return entries;
}
