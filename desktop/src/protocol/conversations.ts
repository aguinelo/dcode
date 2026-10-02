// The list of conversations the daemon keeps (N3), read at the boundary the
// way every event is: a frame that does not match the generated type is a
// fact to say, never a value to guess from.

import type * as P from './generated';
import { check, isRecord, opt, req, type Shape } from './validate';

const CONVERSATION: Shape<P.Conversation> = {
  id: req('string'),
  title: req('string'),
  named: opt('boolean'),
  workspace: req('string'),
  branch: opt('string'),
  model: opt('string'),
  state: req('string'),
  live: req('boolean'),
  turns: req('number'),
  verification: opt('string'),
  added: opt('number'),
  removed: opt('number'),
  files: opt('number'),
  started: req('string'),
  last_activity: req('string'),
  last_event: opt('string'),
  continued_from: opt('string'),
};

/** One frame of the list's stream, read. */
export type ListChange =
  | { kind: 'snapshot'; conversations: P.Conversation[] }
  | { kind: 'changed'; conversation: P.Conversation }
  | { kind: 'removed'; id: string };

/** The kinds this desktop reads, held to the generated constants by a test. */
export const LIST_CHANGE_KINDS = ['snapshot', 'changed', 'removed'] as const;

export type DecodedChange = { ok: true; change: ListChange } | { ok: false; reason: string };

/** Reads one frame of `GET /v1/conversations/events`, as it came off the wire. */
export function decodeChange(raw: unknown): DecodedChange {
  const fail = (reason: string): DecodedChange => ({ ok: false, reason: `lista de conversas: ${reason}` });
  if (!isRecord(raw)) return fail('a mudança não é um objeto');
  switch (raw.kind) {
    case 'snapshot': {
      // An empty list leaves the field out (omitempty), and Go writes a nil
      // slice as null: both are the empty list.
      const list = raw.conversations ?? [];
      if (!Array.isArray(list)) return fail('conversations deveria ser uma lista');
      for (let i = 0; i < list.length; i++) {
        const why = check(CONVERSATION, list[i], `conversations[${i}]`);
        if (why) return fail(why);
      }
      return { ok: true, change: { kind: 'snapshot', conversations: list as P.Conversation[] } };
    }
    case 'changed': {
      const why = check(CONVERSATION, raw.conversation, 'conversation');
      if (why) return fail(why);
      return { ok: true, change: { kind: 'changed', conversation: raw.conversation as P.Conversation } };
    }
    case 'removed':
      if (typeof raw.id !== 'string' || raw.id === '') return fail('removed sem o id da conversa');
      return { ok: true, change: { kind: 'removed', id: raw.id } };
    default:
      return fail(`tipo de mudança que esta versão não conhece: ${String(raw.kind)}`);
  }
}
