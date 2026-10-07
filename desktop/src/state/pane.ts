// What one panel of the grid says about its conversation, at a glance: what
// it is doing, how its last turn was measured, and the little it takes to
// decide whether to look closer. Pure: `now` comes in.
//
// The measurement leads. A turn's seal and its criteria — met, unmet, could
// not run — are what the harness established; the model's own words come
// after, and only their first line.

import type * as P from '../protocol/generated';
import { activity, toolLine, type Tone } from './flow';
import { age, elapsed, firstLine, plural, since } from './format';
import type { Entry, SessionView } from './session';
import { basename, type Row } from './sidebar';

/** One criterion of the definition of done, as the last measured turn left it. */
export interface Light {
  name: string;
  state: 'met' | 'unmet' | 'unavailable';
}

/** One line of a panel's tail, in the three voices of the terminal: you, the work, the answer. */
export interface PaneLine {
  kind: 'you' | 'work' | 'said' | 'note';
  text: string;
  tone: Tone | 'text';
  /** For the work: the call's state, as the flow draws it. */
  glyph?: string;
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
  lines: PaneLine[];
  /** The question waiting for the person, answered from the panel. */
  approval: P.ApprovalRequest | null;
  where: string;
  when: string;
}

const LINE_LIMIT = 120;

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

function pendingApproval(v: SessionView): P.ApprovalRequest | null {
  if (!v.pendingApprovalId) return null;
  for (let i = v.entries.length - 1; i >= 0; i--) {
    const e = v.entries[i];
    if (e?.kind === 'approval' && e.request.approval_id === v.pendingApprovalId) return e.request;
  }
  return null;
}

/** How many of the latest entries a panel reads: more than fit, the oldest cut at the top. */
const TAIL = 14;

/**
 * The end of the conversation, as a terminal shows it: what you asked, the
 * calls, what was answered — newest at the bottom.
 */
function tailOf(v: SessionView): PaneLine[] {
  const out: PaneLine[] = [];
  for (const e of v.entries.slice(-TAIL)) {
    switch (e.kind) {
      case 'user':
        out.push({ kind: 'you', text: firstLine(e.text, LINE_LIMIT), tone: 'text' });
        break;
      case 'tool': {
        const t = toolLine(e.call);
        out.push({ kind: 'work', glyph: t.glyph, tone: t.glyphTone, text: `${t.name} ${firstLine(t.target, LINE_LIMIT)}`.trim() });
        break;
      }
      case 'model':
        out.push({ kind: 'said', text: e.text.trim(), tone: 'text' });
        break;
      case 'error':
        out.push({ kind: 'note', text: firstLine(e.message, LINE_LIMIT), tone: 'err' });
        break;
      default:
    }
  }
  if (v.state === 'running' && activity(v.entries).fact.kind === 'none') out.push({ kind: 'note', text: 'Pensando…', tone: 'dim' });
  if (out.length === 0) out.push({ kind: 'note', text: 'Nada perguntado ainda.', tone: 'dim' });
  return out;
}

function linesOf(r: Row, v: SessionView | undefined): PaneLine[] {
  if (v) return tailOf(v);
  if (r.state === 'recorded') {
    return [{ kind: 'note', text: `Terminada · ${plural(r.turns ?? 0, 'turno', 'turnos')}. Abrir continua numa sessão nova.`, tone: 'dim' }];
  }
  return [{ kind: 'note', text: 'Parada, esperando mensagem.', tone: 'dim' }];
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
    lines: linesOf(r, v),
    approval: v ? pendingApproval(v) : null,
    where,
    when,
  };
}
