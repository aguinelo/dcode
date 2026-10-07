// What a window of the grid says of its conversation above the flow, and what
// a maximized one says beside its peers: what it is doing and how its last
// turn was measured. Pure: `now` comes in.
//
// The measurement leads. A turn's seal and its criteria — met, unmet, could
// not run — are what the harness established, not what the model said.

import type * as P from '../protocol/generated';
import type { Tone } from './flow';
import { age, elapsed, plural, since } from './format';
import type { Entry, SessionView } from './session';
import { basename, type Row } from './sidebar';

/** One criterion of the definition of done, as the last measured turn left it. */
export interface Light {
  name: string;
  state: 'met' | 'unmet' | 'unavailable';
}

export interface PaneView {
  glyph: string;
  tone: Tone;
  /** Working right now: the glyph breathes. */
  live: boolean;
  /** The seal of the last measured turn (D13), when there is one. */
  seal: { text: string; tone: Tone } | null;
  /** Each criterion, when the last measured turn names them. */
  lights: Light[];
  /** Where the work is measured, written in the last measured turn: never left out. */
  touched: string[];
  /** What is known about done when nothing was measured yet. */
  doneNote: string | null;
  where: string;
  when: string;
}

function lastCompletion(entries: readonly Entry[]): P.Completion | null {
  for (let i = entries.length - 1; i >= 0; i--) {
    const e = entries[i];
    if (e?.kind === 'note' && e.note.kind === 'completion') return e.note.completion;
  }
  return null;
}

/** The seal in a word, as the list and the turn both carry it (D13). */
export function sealWord(verification: string | undefined): { text: string; tone: Tone } | null {
  switch (verification) {
    case 'passed':
      return { text: '✓ verificado', tone: 'ok' };
    case 'failed':
      return { text: '✗ não verificado', tone: 'err' };
    case 'stale':
      return { text: '⚠ não conferido · mudou depois do check', tone: 'dim' };
    case 'unavailable':
      return { text: '⚠ não conferido · nada pôde ser conferido', tone: 'dim' };
    default:
      return null;
  }
}

function glyphOf(state: string, verification: string | undefined): { glyph: string; tone: Tone } {
  if (state === 'blocked') return { glyph: '◆', tone: 'warn' };
  if (state === 'running') return { glyph: '●', tone: 'accent' };
  if (verification === 'passed') return { glyph: '✓', tone: 'ok' };
  if (verification === 'failed') return { glyph: '✗', tone: 'err' };
  if (verification === 'stale' || verification === 'unavailable') return { glyph: '⚠', tone: 'dim' };
  return { glyph: '·', tone: 'faint' };
}

export function paneView(r: Row, v: SessionView | undefined, now: number): PaneView {
  const completion = v ? lastCompletion(v.entries) : null;
  const verification = completion?.verification ?? r.verification;
  const state = v?.state ?? r.state;
  const touched = completion?.touched_protected ?? [];
  const mark = glyphOf(state, verification);
  // A pass measured with a ruler the turn changed is not a plain one.
  const { glyph, tone } = touched.length > 0 && mark.tone === 'ok' ? { ...mark, tone: 'warn' as const } : mark;
  const lights: Light[] = completion
    ? [
        ...(completion.met ?? []).map((name) => ({ name, state: 'met' as const })),
        ...(completion.unmet ?? []).map((name) => ({ name, state: 'unmet' as const })),
        ...(completion.unavailable ?? []).map((name) => ({ name, state: 'unavailable' as const })),
      ]
    : [];
  const declared = v?.info?.done_criteria ?? 0;
  const doneNote =
    lights.length > 0 || sealWord(verification)
      ? null
      : declared > 0
        ? `${plural(declared, 'critério', 'critérios')} · medidos ao fim do turno`
        : 'sem definição de pronto';
  const branch = v?.info?.branch || r.branch;
  const where = [basename(r.workspace), branch].filter(Boolean).join(' · ');
  const turnStart = v?.turn?.startedAt;
  const when =
    state === 'running' && turnStart
      ? elapsed(since(turnStart, now))
      : [r.lastAt ? age(since(r.lastAt, now)) : '', plural(v?.turns ?? r.turns ?? 0, 'turno', 'turnos')].filter(Boolean).join(' · ');
  return {
    glyph,
    tone,
    live: state === 'running',
    seal: lights.length > 0 ? null : sealWord(verification),
    lights,
    touched,
    doneNote,
    where,
    when,
  };
}
