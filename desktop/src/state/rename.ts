// Renaming a conversation from its title (D43): the field that opens on a
// double-click, or from "Renomear" in the ⋯ menu. Pure — what is typed and
// what the daemon answered are passed in.
//
// The title is never changed here. It comes back from the daemon, in the
// session's `session.renamed` or in the list's row, so the window shows the
// name the daemon kept rather than the one typed.

/** The field open on one conversation's title. */
export interface Renaming {
  id: string;
  draft: string;
  /** Sent, and the daemon has not answered yet. */
  saving: boolean;
  /** What the daemon last refused, kept in the field for the person to fix. */
  refused: string | null;
}

/** What to do when the person presses Enter or leaves the field. */
export type Commit = { kind: 'send'; name: string } | { kind: 'close' } | { kind: 'stay' };

export function beginRename(id: string, title: string): Renaming {
  return { id, draft: title, saving: false, refused: null };
}

export function editRename(r: Renaming, draft: string): Renaming {
  return { ...r, draft };
}

/**
 * Enter and leaving the field both save, and Esc (handled by the field) does
 * not. Nothing changed closes without asking anyone; an empty field is a
 * name too — it gives the automatic title back. While an answer is on its
 * way nothing is sent twice, and leaving a field the daemon just refused does
 * not send the same text again: the field stays, with the text, until the
 * person changes it, asks again with Enter, or gives up with Esc.
 */
export function commitRename(r: Renaming, title: string, how: 'enter' | 'blur'): Commit {
  if (r.saving) return { kind: 'stay' };
  const name = r.draft.trim();
  if (name === title.trim()) return { kind: 'close' };
  if (how === 'blur' && r.refused === r.draft) return { kind: 'stay' };
  return { kind: 'send', name };
}

export function sentRename(r: Renaming): Renaming {
  return { ...r, saving: true };
}

export function refusedRename(r: Renaming): Renaming {
  return { ...r, saving: false, refused: r.draft };
}
