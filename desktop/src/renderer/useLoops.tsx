import { createContext, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { StopAfterSpec, StopNow, type LoopRun } from '../protocol/generated';
import type { Attention } from '../state/attention';
import { carryHolds, liveLoopIn, loopNews, loopOfSession, type LoopsState } from '../state/loop';
import { LoopSheet } from './LoopSheet';
import type { Outcome, WindowActions } from './window';

type Once = <T>(key: string, run: () => Promise<Outcome<T>>) => Promise<Outcome<T> | null>;

/** What a window reaches about loops (D55) without every component in between carrying it. */
export interface LoopsView {
  /** The loop a conversation belongs to, if it does. */
  of: (sessionId: string) => LoopRun | null;
  /** `/loop` in a field: starts one in the project, a refusal said with the daemon's reason. */
  start: (workspace: string, args: string) => Promise<boolean>;
  stopNow: (loopId: string) => void;
  stopAfter: (loopId: string) => void;
  /** Opens a folder's session, ended or not, where it can be read (D42). */
  openSession: (sessionId: string) => void;
}

export const LoopContext = createContext<LoopsView | null>(null);

const STARTED = 'O loop começou: a janela dele aparece na grade quando a primeira sessão abrir.';

/**
 * The loops of the window (D55): starting one from the field or the sheet,
 * stopping one, and what is said on its own — a run that fails. A window the
 * person held keeps being held as its loop moves to the next folder.
 */
export function useLoops(o: {
  loops: LoopsState;
  actions: WindowActions;
  once: Once;
  say: (text: string) => void;
  updateAttention: (fn: (a: Attention) => Attention) => void;
  show: (id: string | null) => void;
}): { view: LoopsView; sheet: (workspace: string) => void; node: ReactNode; open: boolean } {
  const { loops, actions, once, say, updateAttention, show } = o;
  const [sheetFor, setSheetFor] = useState<string | null>(null);
  const before = useRef(loops);

  useEffect(() => {
    for (const text of loopNews(before.current, loops)) say(text);
    before.current = loops;
    updateAttention((a) => carryHolds(a, loops));
  }, [loops, say, updateAttention]);

  const start = useCallback(
    async (workspace: string, args: string) => {
      const out = await once(`loop:${workspace}`, () => actions.startLoop(workspace, args));
      if (out?.ok) say(STARTED);
      return !!out?.ok;
    },
    [actions, once, say],
  );

  const view = useMemo<LoopsView>(
    () => ({
      of: (id) => loopOfSession(loops, id),
      start,
      stopNow: (id) => void once(`stop-loop:${id}`, () => actions.stopLoop(id, StopNow)),
      stopAfter: (id) => void once(`stop-loop:${id}`, () => actions.stopLoop(id, StopAfterSpec)),
      openSession: (id) => show(id),
    }),
    [actions, loops, once, show, start],
  );

  const close = useCallback(() => setSheetFor(null), []);
  const node = sheetFor ? (
    <LoopSheet
      workspace={sheetFor}
      live={liveLoopIn(loops, sheetFor)}
      listSpecs={actions.listSpecs}
      onStart={async (args) => {
        const out = await actions.startLoop(sheetFor, args);
        if (out.ok) say(STARTED);
        return out;
      }}
      onClose={close}
    />
  ) : null;

  return { view, sheet: setSheetFor, node, open: sheetFor !== null };
}
