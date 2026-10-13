import { useCallback, useState } from 'react';
import { nextMode } from '../state/mode';
import type { SessionsState } from '../state/sessions';
import type { Outcome, WindowActions } from './window';

type Once = <T>(key: string, run: () => Promise<Outcome<T>>) => Promise<Outcome<T> | null>;

/**
 * The mode menu of one session and ⇧⇥ (D36). Both ask the daemon and change
 * nothing here: the chip follows the session's `session.mode_changed`, and a
 * refusal is said by `once` with the daemon's reason.
 */
export function useModeMenu(actions: WindowActions, sessions: SessionsState, once: Once, say: (text: string) => void) {
  const [menu, setMenu] = useState<{ id: string; anchor: HTMLElement } | null>(null);

  const switchTo = useCallback(
    (id: string, mode: string) => void once(`mode:${id}`, () => actions.setMode(id, mode)),
    [actions, once],
  );

  const open = useCallback(
    (id: string, anchor: HTMLElement) => setMenu((m) => (m?.id === id ? null : { id, anchor })),
    [],
  );

  const pick = useCallback(
    (mode: string) => {
      const id = menu?.id;
      setMenu(null);
      if (id) switchTo(id, mode);
    },
    [menu, switchTo],
  );

  /** ⇧⇥: the next mode, from the one the session announced last. */
  const cycle = useCallback(
    (id: string) => {
      const s = sessions.byId[id];
      if (!s?.info) {
        say('Esta conversa não tem sessão aberta: o modo muda numa conversa viva.');
        return;
      }
      switchTo(id, nextMode(s.mode));
    },
    [sessions, say, switchTo],
  );

  const close = useCallback(() => setMenu(null), []);
  const session = menu ? sessions.byId[menu.id] : undefined;
  return { menu, session, open, pick, cycle, close };
}
