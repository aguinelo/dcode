import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { History } from '../state/history';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import type { WindowActions } from './window';

/**
 * The histories of the conversations on screen that ended and are not live
 * (D42) — of `drawn`, the ones the list says are recorded and whose events
 * this window does not hold. Each
 * read once, when it first shows, and kept while the window is open — a
 * recorded conversation does not change. A reading that fails is said in a
 * notice as well as where the thread would be, and is read again on request.
 */
export function useHistories(
  actions: Pick<WindowActions, 'readHistory'>,
  drawn: readonly string[],
  rows: ReadonlyMap<string, Row>,
  sessions: SessionsState,
  say: (text: string) => void,
): {
  of: (id: string) => History | undefined;
  retry: (id: string) => void;
  ended: (id: string) => boolean;
  /** What a maximized ended conversation draws with: its history, a retry, and continuing it. */
  recorded: (id: string, onContinue: (text: string) => void) => { history: History | undefined; onRetry: () => void; onContinue: (text: string) => void };
} {
  const ended = useCallback((id: string) => rows.get(id)?.state === 'recorded' && !sessions.byId[id], [rows, sessions]);
  // Keyed by the ids, so a list that changes elsewhere does not ask again.
  const key = drawn.filter(ended).join(' ');
  const shown = useMemo(() => (key ? key.split(' ') : []), [key]);
  const [all, setAll] = useState<Readonly<Record<string, History>>>({});
  const asked = useRef(new Set<string>());

  const read = useCallback(
    (id: string) => {
      asked.current.add(id);
      setAll((a) => ({ ...a, [id]: { state: 'loading' } }));
      actions.readHistory(id).then(
        (h) => {
          if (h.state === 'failed') say(h.reason);
          setAll((a) => ({ ...a, [id]: h }));
        },
        (err: unknown) => {
          const reason = `O histórico desta conversa não foi lido: ${(err as Error).message ?? String(err)}`;
          say(reason);
          setAll((a) => ({ ...a, [id]: { state: 'failed', reason } }));
        },
      );
    },
    [actions, say],
  );

  useEffect(() => {
    for (const id of shown) if (!asked.current.has(id)) read(id);
  }, [shown, read]);

  return {
    of: useCallback((id: string) => all[id], [all]),
    retry: read,
    ended,
    recorded: (id, onContinue) => ({ history: all[id], onRetry: () => read(id), onContinue }),
  };
}
