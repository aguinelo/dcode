import { useEffect, useRef, useState } from 'react';
import type { Outcome } from './window';

/** How long the orchestrator's stream stays quiet before its list is read again. */
export const SETTLE_MS = 1500;

/**
 * Reads an orchestrator's children and questions from the daemon (D64, D66):
 * at once when its conversation opens, and again each time its stream goes
 * quiet. Once, because the events replayed on opening may still be arriving
 * when the first answer does; the daemon's answer is what holds the
 * retention — a closed child it no longer carries leaves — and the quiet read
 * is what keeps the list on it, rather than a clock of the window's. Answers
 * why the last read failed, for the panel to say.
 */
export function useAgentsRead(id: string | null, orchestrator: boolean, lastSeq: number, read: (id: string) => Promise<Outcome<null>>): string | null {
  const [failed, setFailed] = useState<{ id: string; why: string } | null>(null);
  const readFor = useRef<string | null>(null);
  useEffect(() => {
    if (!id || !orchestrator) return;
    const first = readFor.current !== id;
    readFor.current = id;
    let current = true;
    const timer = setTimeout(
      () =>
        void read(id).then((out) => {
          if (current) setFailed(out.ok ? null : { id, why: out.why });
        }),
      first ? 0 : SETTLE_MS,
    );
    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [id, orchestrator, lastSeq, read]);
  return failed && failed.id === id ? failed.why : null;
}
