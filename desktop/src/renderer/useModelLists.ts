import { useEffect, useRef, useState } from 'react';
import type { ModelsResponse } from '../protocol/generated';
import type { WindowActions } from './window';

/**
 * The daemon's list of models for each project on the stage, asked once per
 * project while the window is open: it is what lets a panel say its model's
 * family was never measured. A project whose list did not come marks nothing,
 * which is "not known" and never "measured" — the menu, opened there, says why.
 */
export function useModelLists(actions: WindowActions, workspaces: readonly string[]): ReadonlyMap<string, ModelsResponse> {
  const [lists, setLists] = useState<ReadonlyMap<string, ModelsResponse>>(() => new Map());
  const asked = useRef(new Set<string>());
  useEffect(() => {
    for (const ws of workspaces) {
      if (!ws || asked.current.has(ws)) continue;
      asked.current.add(ws);
      void actions.listModels(ws).then((out) => {
        if (out.ok) setLists((m) => new Map(m).set(ws, out.value));
      });
    }
  }, [actions, workspaces]);
  return lists;
}
