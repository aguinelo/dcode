import { useCallback, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import { hold, markSeen, release, type Attention } from '../state/attention';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import { markSent, moveDraft, refuse, STEER_TAKES_TEXT } from '../state/attachments';
import { compactText, parseInput, undoText } from '../state/commands';
import { ENDED } from '../state/end';
import { useImageStore } from './useImages';

// The window's pictures are kept with its windows (D46); App provides them to the fields.
export { ImagesProvider } from './useImages';
import type { Sheet } from './CommandSheet';
import { listOf, type UserCommands } from './useUserCommands';
import type { Outcome, StartOptions, WindowActions } from './window';

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
  /** Opens the help, or a conversation's plan, over the window (D48). */
  sheet: (s: Sheet) => void;
  /** The commands each project wrote, as the daemon listed them (D50). */
  commandsOf: (workspace: string) => UserCommands;
}) {
  const { actions, rowById, sessions, updateAttention, once, say, setDrafts, maximized, maximize, sheet, commandsOf } = o;
  // The pictures of the fields (D46): attached per conversation, sent with its next message.
  const images = useImageStore(actions, say);
  const attached = images.store;
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

  /** A session opened straight in a project, or from the folder picker with null — no chooser window (Crew, D33). */
  const startIn = useCallback(
    async (workspace: string | null, options?: StartOptions) => {
      let id: string | null = null;
      if (workspace) {
        const created = await once(`new:${workspace}`, () => actions.newSessionIn(workspace, options));
        if (created?.ok) id = created.value;
      } else {
        id = await fromFolder();
      }
      if (id) place(id);
    },
    [actions, fromFolder, once, place],
  );

  const dropChooser = useCallback((key: string) => {
    setChoosers((c) => c.filter((k) => k !== key));
    setOpening((w) => without(w, key));
  }, []);

  /** A chooser answered: the session opens there and takes the chooser's window. */
  const openIn = useCallback(
    async (key: string, workspace: string | null, options?: StartOptions) => {
      setOpening((w) => ({ ...w, [key]: workspace ?? 'outra pasta' }));
      let id: string | null = null;
      if (workspace) {
        const created = await actions.newSessionIn(workspace, options);
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

  /** Desfazer o último turno: what went back and what did not is said (D40). */
  const undoIn = useCallback(
    async (id: string): Promise<boolean> => {
      const out = await once(`undo:${id}`, () => actions.undo(id));
      if (out?.ok) say(undoText(out.value));
      return !!out?.ok;
    },
    [actions, once, say],
  );

  /**
   * Encerrar sessão (D51): the daemon closes it and keeps its record, and the
   * list says it ended; the window says where the conversation went. A refusal
   * is said by once, with the daemon's reason.
   */
  const endIn = useCallback(
    async (id: string): Promise<boolean> => {
      const out = await once(`close:${id}`, () => actions.closeSession(id));
      if (out?.ok) say(ENDED);
      return !!out?.ok;
    },
    [actions, once, say],
  );

  /** Resumir o contexto: said even when there was nothing to compact, which no event says (D40). */
  const compactIn = useCallback(
    async (id: string): Promise<boolean> => {
      const out = await once(`compact:${id}`, () => actions.compact(id));
      if (out?.ok) say(compactText(out.value));
      return !!out?.ok;
    },
    [actions, once, say],
  );

  /**
   * What was typed in a conversation's field (D40): a message for the model,
   * `!` a command run in the session, `/undo`, `/compact` and `/mode <m>` for
   * the harness; `/help` and `/plan` open over the window, and `/init` and
   * `/plan <text>` send the TUI's turn (D48). Any other slash is said to be
   * unknown and stays in the field.
   */
  const sendIn = useCallback(
    async (id: string, text: string) => {
      const workspace = sessions.byId[id]?.info?.workspace ?? '';
      const mine = commandsOf(workspace);
      const input = parseInput(text, listOf(mine));
      const state = sessions.byId[id]?.state;
      let ok: boolean;
      switch (input.kind) {
        case 'unknown':
          // A command the project wrote may be the one that did not come: said with why.
          say(mine.state === 'failed' ? `${input.why} Os comandos deste projeto não vieram: ${mine.why}` : input.why);
          return;
        case 'bad':
          say(input.why);
          return;
        case 'help':
          sheet({ kind: 'help', workspace });
          ok = true;
          break;
        case 'plan':
          sheet({ kind: 'plan', id });
          ok = true;
          break;
        case 'turn':
          // A new turn, not a correction of this one: sent mid-turn it would
          // steer the work under way with a whole other request.
          if (state === 'running' || state === 'blocked') {
            say(`/${input.name} começa um turno: pare o que está rodando ou espere ele terminar.`);
            return;
          }
          ok = !!(await once(`send:${id}`, () => actions.send(id, input.text, false)))?.ok;
          break;
        case 'undo':
          ok = await undoIn(id);
          break;
        case 'compact':
          ok = await compactIn(id);
          break;
        case 'mode':
          ok = !!(await once(`mode:${id}`, () => actions.setMode(id, input.mode)))?.ok;
          break;
        case 'exec':
          ok = !!(await once(`exec:${id}`, () => actions.exec(id, input.command)))?.ok;
          break;
        case 'message': {
          const running = sessions.byId[id]?.state === 'running';
          // The pictures attached go with it (D46): to a turn, never to a correction.
          const images = attached.get().draft[id] ?? [];
          if (running && images.length > 0) {
            say(STEER_TAKES_TEXT);
            attached.update((a) => refuse(a, id, STEER_TAKES_TEXT));
            return;
          }
          const after = sessions.byId[id]?.lastSeq ?? 0;
          const out = await once(`send:${id}`, () => (images.length > 0 ? actions.send(id, text, running, images) : actions.send(id, text, running)));
          ok = !!out?.ok;
          if (out && images.length > 0) attached.update((a) => (out.ok ? markSent(a, id, after, images) : refuse(a, id, out.why)));
        }
      }
      // Emptied only once the daemon took it, and only if what is in the field
      // is still what was sent: a refusal keeps the text, said why.
      if (!ok) return;
      setDrafts((d) => (d[id] === text ? { ...d, [id]: '' } : d));
      updateAttention((a) => hold(a, id));
    },
    [actions, attached, commandsOf, compactIn, once, say, sessions, setDrafts, sheet, undoIn, updateAttention],
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
      // A command acts on a live session; this one ended, and continuing it on a command would start one for nothing.
      const kind = parseInput(text).kind;
      // The help is about the window, and opens from any field (D48).
      if (kind === 'help') return sheet({ kind: 'help' });
      if (kind !== 'message') {
        say('Esta conversa terminou: comandos valem numa conversa viva. Escreva uma mensagem para continuá-la.');
        return;
      }
      const opened = await once(`open:${row.id}`, () => actions.open(row));
      if (!opened?.ok) return;
      const id = opened.value;
      updateAttention((a) => hold(release(a, row.id, row.lastAt), id));
      setDrafts((d) => ({ ...d, [row.id]: '', [id]: text }));
      // The pictures attached go with the text, to the session that continues it (D46).
      attached.update((a) => moveDraft(a, row.id, id));
      setFocusId(id);
      await sendIn(id, text);
    },
    [actions, attached, once, say, sendIn, setDrafts, sheet, updateAttention],
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
    images,
    focusId,
    setFocusId,
    caret,
    caretTo,
    choosers,
    opening,
    focus,
    open,
    newSession,
    startIn,
    dropChooser,
    openIn,
    sendIn,
    undoIn,
    compactIn,
    endIn,
    stopIn,
    answerIn,
    continueIn,
    replace,
    closer,
  };
}
