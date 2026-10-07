import { useCallback, useState } from 'react';
import { menuOf, type ModelOption } from '../state/models';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import type { Outcome, WindowActions } from './window';

/** The model menu of one conversation: where it opened from, and the daemon's list once it answers. */
export interface ModelMenuState {
  row: Row;
  anchor: HTMLElement;
  options: ModelOption[] | null;
  problem: string | null;
}

type Once = <T>(key: string, run: () => Promise<Outcome<T>>) => Promise<Outcome<T> | null>;

/**
 * The model menu (D28): opened from a conversation's model — a window's chip or
 * the maximized conversation's — filled with what the daemon says it can
 * continue on, and the pick that continues it. `switched` hands the new session
 * the place of the one it continues, which the daemon has closed.
 */
export function useModelMenu(actions: WindowActions, sessions: SessionsState, once: Once, switched: (from: Row, to: string) => void) {
  const [menu, setMenu] = useState<ModelMenuState | null>(null);
  const workspaceOf = useCallback((row: Row) => sessions.byId[row.id]?.info?.workspace ?? row.workspace, [sessions]);

  const open = useCallback(
    (row: Row, anchor: HTMLElement) => {
      if (menu?.row.id === row.id) {
        setMenu(null);
        return;
      }
      const info = sessions.byId[row.id]?.info;
      // The endpoint is known from the session's own facts; the list has only the model.
      const running = info ? { model: info.model, baseUrl: info.base_url ?? '' } : row.model ? { model: row.model, baseUrl: null } : null;
      setMenu({ row, anchor, options: null, problem: null });
      void actions.listModels(workspaceOf(row)).then((listed) =>
        // For the menu still open on this conversation; a closed or moved one takes nothing.
        setMenu((m) =>
          m?.row.id !== row.id ? m : listed.ok ? { ...m, options: menuOf(listed.value, running) } : { ...m, problem: listed.why },
        ),
      );
    },
    [actions, menu, sessions, workspaceOf],
  );

  const pick = useCallback(
    async (name: string) => {
      const row = menu?.row;
      setMenu(null);
      if (!row) return;
      const out = await once(`model:${row.id}`, () => actions.switchModel(row.id, workspaceOf(row), name));
      if (out?.ok) switched(row, out.value);
    },
    [actions, menu, once, switched, workspaceOf],
  );

  const close = useCallback(() => setMenu(null), []);
  // What the session's own events say first: the list's state can lag a turn that just started.
  const state = menu ? (sessions.byId[menu.row.id]?.state ?? menu.row.state) : null;
  return { menu, open, pick, close, busy: state === 'running' || state === 'blocked' };
}
