// What a running turn says of itself against its ceilings, and what a finished
// one cost (D44). Pure, read from the folded session.
//
// The ceilings are drawn on the TUI's terms (internal/tui/statusbar.go,
// ceilingSegment): the rounds from half their ceiling, the calls in flight
// when every slot is taken. Far from a ceiling the count is a number nobody
// acts on, and a row of numbers nobody acts on is a row people stop reading —
// so it stays in the detail, read on hover, and the window draws nothing new.

import type * as P from '../protocol/generated';
import type { Counter, SessionView } from './session';

export type CeilingTone = 'warn' | 'err';

export interface Ceiling {
  text: string;
  tone: CeilingTone;
}

export interface TurnTelemetry {
  /** Drawn only from half the ceiling; err from three quarters. */
  rounds: Ceiling | null;
  /** Drawn only when every slot is taken. */
  inFlight: Ceiling | null;
  /** Every count the daemon sent, said whole, for the tooltip and the Painel. */
  detail: string;
}

function roundsCeiling(c: Counter | null): Ceiling | null {
  if (!c || c.total === null || c.done * 2 < c.total) return null;
  return { text: `rodada ${c.done}/${c.total}`, tone: c.done * 4 >= c.total * 3 ? 'err' : 'warn' };
}

function inFlightCeiling(c: Counter | null): Ceiling | null {
  if (!c || c.total === null || c.done < c.total) return null;
  return { text: `${c.done}/${c.total} em voo`, tone: 'warn' };
}

function roundsDetail(c: Counter): string {
  return c.total === null ? `rodada ${c.done}` : `rodada ${c.done} de ${c.total}`;
}

function inFlightDetail(c: Counter): string {
  const calls = c.done === 1 ? 'chamada' : 'chamadas';
  return c.total === null ? `${c.done} ${calls} em voo` : `${c.done} de ${c.total} ${calls} em voo`;
}

/** Null when no turn runs: at rest the window says nothing of a turn. */
export function turnTelemetry(view: SessionView): TurnTelemetry | null {
  if (!view.turn || (view.state !== 'running' && view.state !== 'blocked')) return null;
  const parts: string[] = [];
  if (view.rounds) parts.push(roundsDetail(view.rounds));
  if (view.inFlight) parts.push(inFlightDetail(view.inFlight));
  return {
    rounds: roundsCeiling(view.rounds),
    inFlight: inFlightCeiling(view.inFlight),
    detail: parts.length > 0 ? parts.join(' · ') : 'a primeira rodada ainda não foi contada',
  };
}

/** Tokens as the TUI reads them (`820`, `2,4k`, `1,5M`), with the interface's comma. */
export function tokens(n: number): string {
  if (n < 1000) return String(Math.max(0, n));
  if (n < 1_000_000) return `${(n / 1000).toFixed(1).replace('.', ',')}k`;
  return `${(n / 1_000_000).toFixed(1).replace('.', ',')}M`;
}

/**
 * The output in front; the input — summed across the turn's rounds, which is
 * what it cost and not how full the context is — and the cache behind it.
 */
export function usageFact(u: P.Usage): { text: string; detail: string } {
  const text = `${tokens(u.output_tokens)} de saída`;
  const parts = [text, `${tokens(u.input_tokens)} de entrada, somada pelas rodadas`];
  if (u.cache_read_tokens) parts.push(`${tokens(u.cache_read_tokens)} lidos do cache`);
  if (u.cache_write_tokens) parts.push(`${tokens(u.cache_write_tokens)} escritos no cache`);
  return { text, detail: parts.join(' · ') };
}
