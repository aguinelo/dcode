// What the turn under way touched, for its activity line (D58): the files its
// writing tools were given — read from the calls the flow holds, as the Crew's
// panel reads them (context.ts) — and the pictures this window sent with its
// message (D46). Pure.

import type { ImageAttachment } from '../shared/api';
import { filesTouchedByCall, type TouchedByCall } from './context';
import type { Entry, SessionView } from './session';

export type TurnFile = TouchedByCall;

export interface TurnTouched {
  files: TurnFile[];
  images: ImageAttachment[];
}

const NOTHING: TurnTouched = { files: [], images: [] };

/**
 * The entries of the turn under way: those at or after the instant it started.
 * The calls carry no turn id once folded, and every entry of an earlier turn
 * came before that turn's turn.completed, which came before this turn.started.
 */
function ofTurn(entries: readonly Entry[], startedAt: string): Entry[] {
  const start = Date.parse(startedAt);
  return entries.filter((e) => Date.parse(e.at) >= start);
}

/** The files the turn under way wrote and the pictures sent with it; nothing outside a turn. */
export function turnTouched(view: Pick<SessionView, 'turn' | 'entries'>, imagesAt: ReadonlyMap<number, ImageAttachment[]>): TurnTouched {
  if (!view.turn) return NOTHING;
  const entries = ofTurn(view.entries, view.turn.startedAt);
  const images = entries.flatMap((e) => (e.kind === 'user' && !e.steer ? (imagesAt.get(e.seq) ?? []) : []));
  return { files: filesTouchedByCall(entries), images };
}

/** The first `room` items, and how many more there are: a "+N" says the rest. */
export function chipsOf<T>(items: readonly T[], room: number): { shown: T[]; more: number } {
  const n = Math.max(0, room);
  return { shown: items.slice(0, n), more: Math.max(0, items.length - n) };
}
