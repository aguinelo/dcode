import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CommandInfo } from '../protocol/generated';
import type { SessionsState } from '../state/sessions';
import type { WindowActions } from './window';

/** What the window knows of a project's commands (D50): still on its way, the list, or why it did not come. */
export type UserCommands = { state: 'loading' } | { state: 'ok'; list: readonly CommandInfo[] } | { state: 'failed'; why: string };

const LOADING: UserCommands = { state: 'loading' };

/**
 * The slash commands the person and each project wrote (`GET /v1/commands`),
 * asked once per project the window holds a conversation of — as the TUI reads
 * them once, at start, so a command does not change meaning halfway. A project
 * whose list did not come keeps the reason, which the / menu and an unknown
 * command say; it is never an empty list.
 */
export function useUserCommands(actions: WindowActions, sessions: SessionsState): (workspace: string) => UserCommands {
  const [known, setKnown] = useState<ReadonlyMap<string, UserCommands>>(() => new Map());
  const asked = useRef(new Set<string>());
  const workspaces = useMemo(
    () => [...new Set(Object.values(sessions.byId).map((v) => v.info?.workspace ?? ''))].filter((w) => w !== ''),
    [sessions],
  );
  useEffect(() => {
    for (const ws of workspaces) {
      if (asked.current.has(ws)) continue;
      asked.current.add(ws);
      void actions.listCommands(ws).then((out) => {
        const got: UserCommands = out.ok ? { state: 'ok', list: out.value.commands } : { state: 'failed', why: out.why };
        setKnown((m) => new Map(m).set(ws, got));
      });
    }
  }, [actions, workspaces]);
  return useCallback((workspace: string) => known.get(workspace) ?? LOADING, [known]);
}

/** The list a project's commands give the field: none while unknown. */
export function listOf(c: UserCommands): readonly CommandInfo[] {
  return c.state === 'ok' ? c.list : [];
}
