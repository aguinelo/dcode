// How numbers and durations read on screen. The words are Portuguese because
// the design's texts are; nothing here decides anything about the session.

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Plural by count, with the count in front: `plural(4, 'arquivo', 'arquivos')` → "4 arquivos". */
export function plural(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}

/**
 * How long ago, as the sidebar says it: `agora`, `12min`, `2h`, `ontem`, `3d`,
 * `1sem`, `1mês`, `2meses`, `1ano`. Durations, never calendar days, so the
 * same session reads the same whatever the time of day.
 */
export function age(ms: number): string {
  if (ms < MINUTE) return 'agora';
  if (ms < HOUR) return `${Math.floor(ms / MINUTE)}min`;
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h`;
  if (ms < 2 * DAY) return 'ontem';
  const days = Math.floor(ms / DAY);
  if (days < 7) return `${days}d`;
  if (days < 30) return `${Math.floor(days / 7)}sem`;
  if (days < 365) {
    const months = Math.floor(days / 30);
    return months === 1 ? '1mês' : `${months}meses`;
  }
  const years = Math.floor(days / 365);
  return years === 1 ? '1ano' : `${years}anos`;
}

/** A waiting clock: `0:48`, `1:35`, `12:03`, `1:02:09`. */
export function clock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, '0');
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${s}` : `${m}:${s}`;
}

/** Elapsed time on the activity line: `42s`, `1m42s`, `15m22s`, `1h02m`. */
export function elapsed(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  if (total < 60) return `${total}s`;
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  if (h > 0) return `${h}h${String(m).padStart(2, '0')}m`;
  return `${m}m${String(total % 60).padStart(2, '0')}s`;
}

/** Milliseconds between two instants, never negative. */
export function since(then: string, now: number): number {
  const t = Date.parse(then);
  return Number.isNaN(t) ? 0 : Math.max(0, now - t);
}

/** The first line of a text, trimmed, at most `limit` characters. */
export function firstLine(text: string, limit = Infinity): string {
  const line = (text.trim().split('\n')[0] ?? '').trim();
  const chars = Array.from(line);
  return chars.length > limit ? `${chars.slice(0, limit).join('').trim()}…` : line;
}
