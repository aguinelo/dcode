// Every conversation the daemon knows, live and recorded, as the stream of its
// list says (N3). The sidebar and the search read this, never one stream per
// session and never the disk (D21).

import type { ListChange } from '../protocol/conversations';
import type { Conversation } from '../protocol/generated';

export interface ConversationsState {
  byId: Readonly<Record<string, Conversation>>;
  /** The first snapshot arrived. Before it, an empty list is not known to be empty. */
  loaded: boolean;
}

export const emptyConversations: ConversationsState = { byId: {}, loaded: false };

/**
 * Folds one frame. A snapshot replaces everything — the stream opens with one,
 * and a reconnection opens with a new one, so nothing has to be stitched.
 */
export function applyListChange(state: ConversationsState, change: ListChange): ConversationsState {
  switch (change.kind) {
    case 'snapshot':
      return { byId: Object.fromEntries(change.conversations.map((c) => [c.id, c])), loaded: true };
    case 'changed':
      return { ...state, byId: { ...state.byId, [change.conversation.id]: change.conversation } };
    case 'removed': {
      if (!(change.id in state.byId)) return state;
      const byId = { ...state.byId };
      delete byId[change.id];
      return { ...state, byId };
    }
  }
}

/**
 * The live conversation that continues `id`, newest first, if one does:
 * opening an ended conversation a second time opens its continuation rather
 * than starting another.
 */
export function liveContinuationOf(state: ConversationsState, id: string): Conversation | null {
  let found: Conversation | null = null;
  for (const c of Object.values(state.byId)) {
    if (!c.live || c.continued_from !== id) continue;
    if (!found || Date.parse(c.started) > Date.parse(found.started)) found = c;
  }
  return found;
}
