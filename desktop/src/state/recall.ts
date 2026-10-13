// What a conversation's field recalls with ↑/↓ (D47), as the TUI's prompt line
// does: what was asked before, newest first, with the draft kept and given back.
//
// Derived from the conversation's own events — turn.started and turn.steered,
// which the daemon replays — so it survives a restart without a second store.
// Pure: the text, the caret and the walk go in, the next walk comes out.

import type { Entry } from './session';

/** Where the walk is: `back` steps behind the newest prompt (0 = not walking), and the draft it began from. */
export interface Recall {
  back: number;
  draft: string;
}

export const notRecalling: Recall = { back: 0, draft: '' };

/** What was asked and steered, oldest first, trimmed; a repeat right after itself counts once. */
export function promptsOf(entries: readonly Entry[]): string[] {
  const out: string[] = [];
  for (const e of entries) {
    if (e.kind !== 'user') continue;
    const text = e.text.trim();
    if (text && out[out.length - 1] !== text) out.push(text);
  }
  return out;
}

/** A continuation's prompts after those of the conversation it continues, the repeat at the seam once. */
export function olderPrompts(older: readonly string[], own: readonly string[]): string[] {
  if (older.length > 0 && own.length > 0 && older[older.length - 1] === own[0]) return [...older, ...own.slice(1)];
  return [...older, ...own];
}

/** The conversation this one continues (session.resumed), or null. */
export function sourceOf(entries: readonly Entry[]): string | null {
  for (const e of entries) if (e.kind === 'note' && e.note.kind === 'resumed') return e.note.sourceId;
  return null;
}

/** ↑ one step: null at the oldest, or with nothing to recall. */
export function recallUp(list: readonly string[], r: Recall, text: string): { recall: Recall; text: string } | null {
  if (r.back >= list.length) return null;
  const back = r.back + 1;
  return { recall: { back, draft: r.back === 0 ? text : r.draft }, text: list[list.length - back] ?? '' };
}

/** ↓ one step: past the newest, the draft the walk began from; null when not walking. */
export function recallDown(list: readonly string[], r: Recall): { recall: Recall; text: string } | null {
  if (r.back === 0) return null;
  const back = Math.min(r.back, list.length) - 1;
  if (back <= 0) return { recall: notRecalling, text: r.draft };
  return { recall: { back, draft: r.draft }, text: list[list.length - back] ?? '' };
}

/**
 * Whether ↑ recalls rather than moves the caret: on the first line, with
 * nothing selected, and the field empty, the caret at its start, or a walk
 * already on. A draft's first ↑ goes to its start, as a field's does.
 */
export function wantsUp(text: string, start: number, end: number, r: Recall): boolean {
  if (start !== end || text.slice(0, start).includes('\n')) return false;
  return text === '' || start === 0 || r.back > 0;
}

/** Whether ↓ comes forward rather than moves the caret: walking, nothing selected, on the last line. */
export function wantsDown(text: string, start: number, end: number, r: Recall): boolean {
  return r.back > 0 && start === end && !text.slice(end).includes('\n');
}
