import { useCallback, useState } from 'react';
import { ATTENTION_KEY, beginAt, emptyAttention, parseAttention, type Attention } from '../state/attention';

/**
 * The grid's memory, kept in localStorage like the sidebar's arrangement (D7).
 * What cannot be read or written is said, and the grid goes on from what it has.
 */
export function useAttention(say: (text: string) => void): [Attention, (fn: (a: Attention) => Attention) => void] {
  const [attention, setAttention] = useState<Attention>(() => {
    let raw: string | null = null;
    try {
      raw = window.localStorage.getItem(ATTENTION_KEY);
    } catch (err) {
      queueMicrotask(() => say(`Não foi possível ler a memória da grade (${(err as Error).message}); ela começa de agora.`));
    }
    const read = parseAttention(raw);
    if (read.problem) queueMicrotask(() => say(`${read.problem}; a grade começa de agora.`));
    const begun = beginAt(read.attention ?? emptyAttention, new Date().toISOString());
    // The first run's instant is kept at once: what finishes before the next
    // launch is news then, not history.
    if (begun !== read.attention) {
      try {
        window.localStorage.setItem(ATTENTION_KEY, JSON.stringify(begun));
      } catch (err) {
        queueMicrotask(() => say(`Não foi possível guardar a memória da grade (${(err as Error).message}); vale só até fechar.`));
      }
    }
    return begun;
  });
  const update = useCallback(
    (fn: (a: Attention) => Attention) => {
      setAttention((current) => {
        const next = fn(current);
        if (next === current) return current;
        try {
          window.localStorage.setItem(ATTENTION_KEY, JSON.stringify(next));
        } catch (err) {
          say(`Não foi possível guardar a memória da grade (${(err as Error).message}); vale só até fechar.`);
        }
        return next;
      });
    },
    [say],
  );
  return [attention, update];
}
