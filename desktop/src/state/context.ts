// What the Crew view's right panel says of the conversation in focus, beside
// its seal and criteria: the files it changed. Read from the calls the flow
// already holds — the path a writing tool was given, and the lines it reported
// (D8) — never from the text of an output.

import type { Entry } from './session';

/** The tools that change a file named by `path`. */
const WRITERS = new Set(['write', 'edit', 'multi_edit', 'apply_patch']);

export interface TouchedFile {
  path: string;
  added: number;
  removed: number;
  /** The last write to it failed. */
  failed: boolean;
}

function pathOf(input: unknown): string | null {
  if (typeof input !== 'object' || input === null) return null;
  const p = (input as Record<string, unknown>).path;
  return typeof p === 'string' && p !== '' ? p : null;
}

/** A file touched, with the call that last wrote it: what opening it shows (D58). */
export interface TouchedByCall extends TouchedFile {
  callId: string;
}

/** Each file written or edited, once, the most recently touched first, with its last call. */
export function filesTouchedByCall(entries: readonly Entry[]): TouchedByCall[] {
  const byPath = new Map<string, TouchedByCall>();
  for (const e of entries) {
    if (e.kind !== 'tool' || !WRITERS.has(e.call.name)) continue;
    const path = pathOf(e.call.input);
    if (!path) continue;
    const r = e.call.result;
    const before = byPath.get(path) ?? { path, added: 0, removed: 0, failed: false, callId: e.call.id };
    byPath.delete(path);
    byPath.set(path, {
      path,
      added: before.added + (r?.added ?? 0),
      removed: before.removed + (r?.removed ?? 0),
      failed: r ? !r.ok : before.failed,
      callId: e.call.id,
    });
  }
  return [...byPath.values()].reverse();
}

/** Each file written or edited, once, the most recently touched first. */
export function filesTouched(entries: readonly Entry[]): TouchedFile[] {
  return filesTouchedByCall(entries).map(({ path, added, removed, failed }) => ({ path, added, removed, failed }));
}
