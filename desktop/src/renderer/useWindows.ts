import { useCallback, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import { hold, markSeen, release, type Attention } from '../state/attention';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import type { Outcome, WindowActions } from './window';

type Once = <T>(key: string, run: () => Promise<Outcome<T>>) => Promise<Outcome<T> | null>;

function without(r: Readonly<Record<string, string>>, key: string): Record<string, string> {
  const rest = { ...r };
  delete rest[key];
  return rest;
}

/**
 * The grid's windows (D31): the one the person is in, where the caret goes,
 * the windows asking where a new session opens, and what a window does —
 * open, write, continue, close. A window the person opens, clicks into or
 * writes in is held on the grid; closing it lets it go, seen.
 */
export function useWindows(o: {
  actions: WindowActions;
  rowById: ReadonlyMap<string, Row>;
  sessions: SessionsState;
  updateAttention: (fn: (a: Attention) => Attention) => void;
  once: Once;
  say: (text: string) => void;
  setDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  /** The conversation maximized, if one is. */
  maximized: string | null;
  /** Maximizes a conversation, or goes back to the grid with null. */
  maximize: (id: string | null) => void;
}) {
  const { actions, rowById, sessions, updateAttention, once, say, setDrafts, maximized, maximize } = o;
  const [focusId, setFocusId] = useState<string | null>(null);
  const [caret, setCaret] = useState<{ id: string; n: number } | null>(null);
  const [choosers, setChoosers] = useState<string[]>([]);
  const [opening, setOpening] = useState<Record<string, string>>({});
  const made = useRef(0);
  const asking = useRef(false);

  const caretTo = useCallback((id: string) => setCaret((c) => ({ id, n: (c?.n ?? 0) + 1 })), []);

  /** The person is in this window: it stays on the grid, and what it had done is seen. */
  const focus = useCallback(
    (id: string) => {
      setFocusId(id);
      const r = rowById.get(id);
      if (r) updateAttention((a) => markSeen(hold(a, id), id, r.lastAt));
    },
    [rowById, updateAttention],
  );

  /** A conversation just opened: held, in focus, the caret in its field — maximized, if one was. */
  const place = useCallback(
    (id: string) => {
      updateAttention((a) => hold(a, id));
      setFocusId(id);
      caretTo(id);
      if (maximized) maximize(id);
    },
    [caretTo, maximize, maximized, updateAttention],
  );

  /** A row of the list: a live conversation is followed, an ended one continued (D21). */
  const open = useCallback(
    async (id: string) => {
      const row = rowById.get(id);
      if (!row) return;
      const opened = await once(`open:${id}`, () => actions.open(row));
      if (opened?.ok) place(opened.value);
    },
    [actions, once, place, rowById],
  );

  /** Asks for a folder and opens a session there; null and nothing said when the person cancels. */
  const fromFolder = useCallback(async (): Promise<string | null> => {
    if (asking.current) return null;
    asking.current = true;
    try {
      const created = await actions.newSession();
      if (!created) return null;
      if (!created.ok) say(created.why);
      return created.ok ? created.value : null;
    } finally {
      asking.current = false;
    }
  }, [actions, say]);

  /**
   * Nova sessão: a window asking where. With no project known yet there is
   * nothing to choose from, and the folder picker comes at once.
   */
  const newSession = useCallback(
    async (known: number) => {
      if (known === 0) {
        const id = await fromFolder();
        if (id) place(id);
        return;
      }
      const key = `new-${++made.current}`;
      setChoosers((c) => [...c, key]);
      setFocusId(key);
      if (maximized) maximize(null);
    },
    [fromFolder, maximize, maximized, place],
  );

  const dropChooser = useCallback((key: string) => {
    setChoosers((c) => c.filter((k) => k !== key));
    setOpening((w) => without(w, key));
  }, []);

  /** A chooser answered: the session opens there and takes the chooser's window. */
  const openIn = useCallback(
    async (key: string, workspace: string | null) => {
      setOpening((w) => ({ ...w, [key]: workspace ?? 'outra pasta' }));
      let id: string | null = null;
      if (workspace) {
        const created = await actions.newSessionIn(workspace);
        if (created.ok) id = created.value;
        else say(created.why);
      } else {
        id = await fromFolder();
      }
      setOpening((w) => without(w, key));
      if (!id) return;
      dropChooser(key);
      place(id);
    },
    [actions, dropChooser, fromFolder, place, say],
  );

  const sendIn = useCallback(
    async (id: string, text: string) => {
      const running = sessions.byId[id]?.state === 'running';
      const done = await once(`send:${id}`, () => actions.send(id, text, running));
      // Emptied only once the daemon took it, and only if what is in the field
      // is still what was sent: a refusal keeps the text, said why.
      if (!done?.ok) return;
      setDrafts((d) => (d[id] === text ? { ...d, [id]: '' } : d));
      updateAttention((a) => hold(a, id));
    },
    [actions, once, sessions, setDrafts, updateAttention],
  );

  const stopIn = useCallback((id: string) => void once(`stop:${id}`, () => actions.stop(id)), [actions, once]);

  const answerIn = useCallback(
    (id: string, approvalId: string, decision: string) => {
      if (approvalId) void once(`answer:${approvalId}`, () => actions.answer(id, approvalId, decision));
    },
    [actions, once],
  );

  /** An ended conversation written to: it continues in a new session, which takes its window, and the text goes there. */
  const continueIn = useCallback(
    async (row: Row, text: string) => {
      const opened = await once(`open:${row.id}`, () => actions.open(row));
      if (!opened?.ok) return;
      const id = opened.value;
      updateAttention((a) => hold(release(a, row.id, row.lastAt), id));
      setDrafts((d) => ({ ...d, [row.id]: '', [id]: text }));
      setFocusId(id);
      await sendIn(id, text);
    },
    [actions, once, sendIn, setDrafts, updateAttention],
  );

  /** A conversation continued elsewhere — on another model — takes over the window of the one it continues. */
  const replace = useCallback(
    (from: Row, to: string) => {
      updateAttention((a) => hold(release(a, from.id, from.lastAt), to));
      setFocusId((f) => (f === from.id ? to : f));
      if (maximized === from.id) maximize(to);
    },
    [maximize, maximized, updateAttention],
  );

  /** ×: the window goes, what it did seen. A session never asked anything is closed, not left open behind it. */
  const close = useCallback(
    (id: string) => {
      const r = rowById.get(id);
      const v = sessions.byId[id];
      updateAttention((a) => release(a, id, r?.lastAt ?? v?.lastAt ?? null));
      if (v && v.state === 'idle' && v.turns === 0 && !v.firstQuestion) void once(`close:${id}`, () => actions.closeSession(id));
      setFocusId((f) => (f === id ? null : f));
    },
    [actions, once, rowById, sessions, updateAttention],
  );

  /** What closes a window: nothing while its conversation runs or waits, since what works keeps its place. */
  const closer = useCallback(
    (id: string) => {
      const state = sessions.byId[id]?.state ?? rowById.get(id)?.state;
      return state === 'running' || state === 'blocked' ? null : () => close(id);
    },
    [close, rowById, sessions],
  );

  return {
    focusId,
    setFocusId,
    caret,
    caretTo,
    choosers,
    opening,
    focus,
    open,
    newSession,
    dropChooser,
    openIn,
    sendIn,
    stopIn,
    answerIn,
    continueIn,
    replace,
    closer,
  };
}
