import { useEffect } from 'react';
import { flushSync } from 'react-dom';
import type { SessionView } from '../state/session';
import type { SessionsState } from '../state/sessions';
import { answerForKey } from './ApprovalCard';

/** Whether keys typed now go into text. A disabled field takes none. */
function isEditable(el: Element | null): boolean {
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) return !el.disabled;
  return !!el && (el as HTMLElement).isContentEditable;
}

/** Focus nowhere in particular: the window's own keys apply, not a control's. */
function focusIsNowhere(): boolean {
  const el = document.activeElement;
  return !el || el === document.body || (el instanceof HTMLTextAreaElement && el.disabled);
}

/**
 * The window's keys (D31). ⌘K searches, ⌘N opens a session, ⌘0 is the grid,
 * ⌘1–9 puts the caret in window n, ⌘↵ maximizes the window in focus and
 * brings it back. With the focus in no field: the arrows move between windows,
 * ↵ puts the caret in the one in focus, and a window waiting for an answer
 * takes 1, 2, 3, esc and ↵ — which denies (D22). Maximized, esc goes back.
 */
export function useWindowKeys(k: {
  /** The windows, in their places: conversations, then choosers. */
  ids: readonly string[];
  cols: number;
  focusId: string | null;
  setFocusId: (id: string) => void;
  caretTo: (id: string) => void;
  sessions: SessionsState;
  /** The conversation maximized, if one is. */
  maximized: SessionView | null;
  /** While the search or a menu is open, its keys are its own. */
  busy: boolean;
  setSearching: (fn: (s: boolean) => boolean) => void;
  newSession: () => void;
  show: (id: string | null) => void;
  answer: (sessionId: string, approvalId: string, decision: string) => void;
  isConversation: (id: string) => boolean;
}) {
  const { ids, cols, focusId, setFocusId, caretTo, sessions, maximized, busy, setSearching, newSession, show, answer, isConversation } = k;
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      const key = e.key.toLowerCase();
      if (mod && key === 'k') {
        e.preventDefault();
        // Drawn and focused before this key's handling ends, so what is typed
        // right after ⌘K lands in the search, not in the window behind it.
        flushSync(() => setSearching((s) => !s));
        return;
      }
      if (busy) return;
      if (mod && key === 'n') {
        e.preventDefault();
        newSession();
        return;
      }
      if (mod && key === '0') {
        e.preventDefault();
        show(null);
        return;
      }
      if (mod && /^[1-9]$/.test(e.key)) {
        const id = ids[Number(e.key) - 1];
        if (!id) return;
        e.preventDefault();
        if (maximized && isConversation(id)) {
          show(id);
          return;
        }
        if (maximized) show(null);
        setFocusId(id);
        caretTo(id);
        return;
      }
      if (mod && e.key === 'Enter') {
        e.preventDefault();
        if (maximized) show(null);
        else if (focusId && isConversation(focusId)) show(focusId);
        return;
      }
      if (mod || isEditable(document.activeElement)) return;
      // ↵ answers only from nowhere in particular: on a focused control it is
      // that control's own ↵ (D22).
      const nowhere = focusIsNowhere();
      if (maximized) {
        if (maximized.pendingApprovalId) {
          if (e.key === 'Enter' && !nowhere) return;
          const decision = answerForKey(e.key);
          if (decision) {
            e.preventDefault();
            answer(maximized.id, maximized.pendingApprovalId, decision);
          }
        } else if (e.key === 'Escape') {
          e.preventDefault();
          show(null);
        }
        return;
      }
      const n = ids.length;
      if (n === 0) return;
      const at = Math.max(0, focusId ? ids.indexOf(focusId) : 0);
      const moves: Record<string, number> = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -cols, ArrowDown: cols };
      const step = moves[e.key];
      if (step !== undefined) {
        e.preventDefault();
        const to = ids[Math.max(0, Math.min(n - 1, at + step))];
        if (to) setFocusId(to);
        return;
      }
      const id = ids[at];
      if (!id) return;
      const waiting = sessions.byId[id]?.pendingApprovalId;
      if (waiting) {
        if (e.key === 'Enter' && !nowhere) return;
        const decision = answerForKey(e.key);
        if (decision) {
          e.preventDefault();
          answer(id, waiting, decision);
        }
        return;
      }
      if (e.key === 'Enter' && nowhere && isConversation(id)) {
        e.preventDefault();
        caretTo(id);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [ids, cols, focusId, setFocusId, caretTo, sessions, maximized, busy, setSearching, newSession, show, answer, isConversation]);
}
