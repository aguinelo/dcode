import {
  ConversationChanged,
  ConversationRemoved,
  ConversationSnapshot,
  SessionStateBlocked,
  SessionStateRunning,
  type Conversation,
} from '../protocol/generated';
import { isRecord } from '../protocol/validate';

// The list of conversations as the main process keeps it, folded from the
// list's stream (D21). Kept here so a window that opens — or reloads — after
// the stream did starts from the whole list, not from whatever changes next.
//
// Only what folding needs is checked: a kind, and an id on each conversation.
// The renderer decodes every frame against the generated types and says what
// does not match; a frame this fold cannot read is passed on all the same.

export type Folded = { ok: true; list: Conversation[] } | { ok: false; problem: string };

function hasId(v: unknown): v is Conversation {
  return isRecord(v) && typeof v.id === 'string' && v.id !== '';
}

/**
 * Applies one frame: a snapshot replaces the list, `changed` puts the
 * conversation in its place — or first, when it is new — and `removed` takes
 * it out. Before the first snapshot there is no list to change, and a change
 * is not taken for one.
 */
export function foldConversations(list: Conversation[] | null, frame: unknown): Folded {
  if (!isRecord(frame) || typeof frame.kind !== 'string') return { ok: false, problem: 'not a change with a kind' };
  switch (frame.kind) {
    case ConversationSnapshot: {
      // Go leaves an empty list out of the frame (omitempty): no field is no conversations.
      const all = frame.conversations ?? [];
      if (!Array.isArray(all) || !all.every(hasId)) return { ok: false, problem: 'a snapshot without a list of conversations with ids' };
      return { ok: true, list: [...all] };
    }
    case ConversationChanged: {
      if (!hasId(frame.conversation)) return { ok: false, problem: 'a change without a conversation with an id' };
      if (list === null) return { ok: false, problem: 'a change before the first snapshot' };
      const changed = frame.conversation;
      const at = list.findIndex((c) => c.id === changed.id);
      return { ok: true, list: at < 0 ? [changed, ...list] : list.map((c, i) => (i === at ? changed : c)) };
    }
    case ConversationRemoved: {
      if (typeof frame.id !== 'string' || frame.id === '') return { ok: false, problem: 'a removal without an id' };
      if (list === null) return { ok: false, problem: 'a removal before the first snapshot' };
      const gone = frame.id;
      return { ok: true, list: list.filter((c) => c.id !== gone) };
    }
    default:
      return { ok: false, problem: `an unknown kind “${frame.kind}”` };
  }
}

/** Live conversations a stopped daemon would cut off: running, or waiting on an approval. */
export function countWorking(list: readonly Conversation[]): number {
  return list.filter((c) => c.live && (c.state === SessionStateRunning || c.state === SessionStateBlocked)).length;
}
