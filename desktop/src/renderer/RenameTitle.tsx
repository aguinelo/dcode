import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { beginRename, commitRename, editRename, refusedRename, sentRename, type Renaming } from '../state/rename';
import type { Outcome } from './window';

// A conversation's title, which turns into a field to rename it (D43): a
// double-click on it, or "Renomear" in the ⋯ menu. Enter and leaving the field
// save, Esc gives up. The title shown is always the daemon's — it changes when
// `session.renamed` or the list says so — and a refusal is said in a notice,
// with the field keeping what was typed.

export interface Renames {
  renaming: Renaming | null;
  /** Changes when the field should take the caret again: after a refusal. */
  focusTick: number;
  begin(id: string, title: string): void;
  edit(draft: string): void;
  commit(title: string, how: 'enter' | 'blur'): void;
  cancel(): void;
}

const RenameContext = createContext<Renames | null>(null);
export const RenameProvider = RenameContext.Provider;

/** The one field open at a time, and what saving it asks of the daemon. */
export function useRenames(rename: (id: string, name: string) => Promise<Outcome<null>>, say: (text: string) => void): Renames {
  const [renaming, setRenaming] = useState<Renaming | null>(null);
  const [focusTick, setFocusTick] = useState(0);
  // Read by the async answer and by a blur that follows Esc in the same tick:
  // both need what the field is now, not what it was when they were bound.
  const now = useRef<Renaming | null>(null);
  const put = useCallback((next: Renaming | null) => {
    now.current = next;
    setRenaming(next);
  }, []);

  const begin = useCallback((id: string, title: string) => put(beginRename(id, title)), [put]);
  const edit = useCallback((draft: string) => now.current && put(editRename(now.current, draft)), [put]);
  const cancel = useCallback(() => put(null), [put]);
  const commit = useCallback(
    (title: string, how: 'enter' | 'blur') => {
      const r = now.current;
      if (!r) return;
      const c = commitRename(r, title, how);
      if (c.kind === 'stay') return;
      if (c.kind === 'close') return put(null);
      put(sentRename(r));
      void rename(r.id, c.name).then((out) => {
        // Another field may have opened meanwhile: only this one's answer closes it.
        const still = now.current?.id === r.id ? now.current : null;
        if (out.ok) {
          if (still) put(null);
          return;
        }
        say(out.why);
        if (still) {
          put(refusedRename(still));
          setFocusTick((t) => t + 1);
        }
      });
    },
    [put, rename, say],
  );

  return useMemo(() => ({ renaming, focusTick, begin, edit, commit, cancel }), [renaming, focusTick, begin, edit, commit, cancel]);
}

/**
 * The title as text, or — while it is being renamed — the field in its place,
 * with the same class so it takes the same room. A double-click on the text
 * opens the field, and is not the double-click of what contains it.
 */
export function ConversationTitle({ id, title, className, tooltip }: { id: string; title: string; className: string; tooltip?: string }) {
  const r = useContext(RenameContext);
  if (r && r.renaming?.id === id) return <TitleField r={r} renaming={r.renaming} title={title} className={className} />;
  return (
    <span
      className={className}
      title={tooltip}
      onDoubleClick={
        r
          ? (e) => {
              e.stopPropagation();
              r.begin(id, title);
            }
          : undefined
      }
    >
      {title}
    </span>
  );
}

function TitleField({ r, renaming, title, className }: { r: Renames; renaming: Renaming; title: string; className: string }) {
  const input = useRef<HTMLInputElement>(null);
  useLayoutEffect(() => {
    input.current?.focus();
    input.current?.select();
  }, []);
  useEffect(() => {
    if (r.focusTick) input.current?.focus();
  }, [r.focusTick]);
  return (
    <input
      ref={input}
      className={`${className} title-field`}
      aria-label="Nome da conversa"
      placeholder="vazio volta ao título automático"
      value={renaming.draft}
      readOnly={renaming.saving}
      aria-busy={renaming.saving || undefined}
      onChange={(e) => r.edit(e.target.value)}
      onBlur={() => r.commit(title, 'blur')}
      onDoubleClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => {
        // The name's keys are the field's: none of the window's apply while it is open.
        e.stopPropagation();
        if (e.key === 'Escape') {
          e.preventDefault();
          r.cancel();
        } else if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
          e.preventDefault();
          r.commit(title, 'enter');
        }
      }}
    />
  );
}
