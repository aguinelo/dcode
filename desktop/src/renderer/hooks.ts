import { useEffect, useState } from 'react';

/** The current time, refreshed every `ms` — what the live clocks count against. */
export function useNow(ms = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), ms);
    return () => window.clearInterval(t);
  }, [ms]);
  return now;
}

function query(q: string): MediaQueryList | null {
  return typeof window.matchMedia === 'function' ? window.matchMedia(q) : null;
}

/** Whether the person asked the system for less motion. */
export function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState(() => query('(prefers-reduced-motion: reduce)')?.matches ?? false);
  useEffect(() => {
    const mq = query('(prefers-reduced-motion: reduce)');
    if (!mq) return;
    const on = () => setReduced(mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, []);
  return reduced;
}

/** A counter that advances every `ms`, or never when `paused`. */
export function useTick(ms: number, paused: boolean): number {
  const [tick, setTick] = useState(0);
  useEffect(() => {
    if (paused) return;
    const t = window.setInterval(() => setTick((n) => n + 1), ms);
    return () => window.clearInterval(t);
  }, [ms, paused]);
  return tick;
}
