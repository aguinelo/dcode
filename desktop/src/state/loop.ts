// The `/loop` runs the daemon conducts (D55), as the window draws them. Pure:
// the runs come in as the daemon sent them, `now` as an argument, and nothing
// here reads a clock, storage or the DOM.
//
// The daemon decides everything about a run — its folders, their phases, the
// verdict. The window only says it, and places it: one window per loop, which
// follows the session the loop is in now (amending D31).

import {
  CodeLoopAlreadyRunning,
  LoopAwaitingReview,
  LoopFailed,
  LoopFinished,
  LoopQualifying,
  LoopStopped,
  LoopSurveying,
  LoopWorking,
  type LoopRun,
} from '../protocol/generated';
import type { LoopListChange } from '../protocol/loops';
import type { Attention } from './attention';
import { elapsed, plural, since } from './format';
import { planGrid, type GridPlan } from './grid';
import type { Row } from './sidebar';

export interface LoopsState {
  byId: Record<string, LoopRun>;
  /** The daemon's order: the oldest first. */
  order: string[];
  /** The first snapshot arrived. */
  loaded: boolean;
  /** Why the daemon gives no loops, when it said so. */
  problem: string | null;
}

export const emptyLoops: LoopsState = { byId: {}, order: [], loaded: false, problem: null };

export function applyLoopChange(s: LoopsState, change: LoopListChange): LoopsState {
  if (change.kind === 'snapshot') {
    const byId: Record<string, LoopRun> = {};
    for (const r of change.loops) byId[r.id] = r;
    return { byId, order: change.loops.map((r) => r.id), loaded: true, problem: null };
  }
  const r = change.loop;
  const order = s.order.includes(r.id) ? s.order : [...s.order, r.id];
  return { ...s, byId: { ...s.byId, [r.id]: r }, order };
}

/** The daemon has no loops to give, and said why. */
export function loopsUnavailable(s: LoopsState, reason: string): LoopsState {
  return { ...s, problem: reason };
}

const LIVE: readonly string[] = [LoopSurveying, LoopQualifying, LoopAwaitingReview, LoopWorking];

export function isLive(r: LoopRun): boolean {
  return LIVE.includes(r.state);
}

function runs(s: LoopsState): LoopRun[] {
  return s.order.flatMap((id) => (s.byId[id] ? [s.byId[id]] : []));
}

/** The loop alive in a project: there is at most one (Q7). */
export function liveLoopIn(s: LoopsState, workspace: string): LoopRun | null {
  return runs(s).find((r) => isLive(r) && r.workspace === workspace) ?? null;
}

/** Every session a run opened, the current one included. */
function sessionsOf(r: LoopRun): string[] {
  const ids = r.specs.flatMap((f) => (f.session_id ? [f.session_id] : []));
  return r.session_id && !ids.includes(r.session_id) ? [...ids, r.session_id] : ids;
}

/** The loop a session belongs to: the one it is current in first, else the newest that opened it. */
export function loopOfSession(s: LoopsState, sessionId: string): LoopRun | null {
  const all = runs(s);
  return all.find((r) => r.session_id === sessionId) ?? all.reverse().find((r) => sessionsOf(r).includes(sessionId)) ?? null;
}

/** For each session a loop opened, the session that loop is in now. */
function currentOf(s: LoopsState): Map<string, string> {
  const to = new Map<string, string>();
  for (const r of runs(s)) {
    if (!r.session_id) continue;
    for (const id of sessionsOf(r)) to.set(id, r.session_id);
  }
  return to;
}

/**
 * Lays out the grid with one window per loop (D55, Q5 A): of a loop's
 * sessions only the current one claims a place, and it takes the place the
 * loop's last one had, so the window stays where it is while the loop moves
 * from folder to folder. A loop whose proposal waits for review waits for you.
 */
export function planWithLoops(rows: readonly Row[], a: Attention, previous: readonly string[], capacity: number, loops: LoopsState): GridPlan {
  const current = currentOf(loops);
  const kept = rows.filter((r) => (current.get(r.id) ?? r.id) === r.id);
  const placed = [...new Set(previous.map((id) => current.get(id) ?? id))];
  const waiting = new Set(runs(loops).flatMap((r) => (r.state === LoopAwaitingReview && r.session_id ? [r.session_id] : [])));
  return planGrid(kept, a, placed, capacity, waiting);
}

/**
 * A window the person held stays held when its loop moves on: the hold goes
 * to the session the loop is in now. The same attention back when nothing
 * moved, so it can be applied on every change.
 */
export function carryHolds(a: Attention, loops: LoopsState): Attention {
  const current = currentOf(loops);
  let pinned = a.pinned;
  for (const id of a.pinned) {
    const to = current.get(id);
    if (!to || to === id) continue;
    pinned = pinned.filter((x) => x !== id);
    if (!pinned.includes(to)) pinned = [...pinned, to];
  }
  return pinned === a.pinned ? a : { ...a, pinned };
}

const STATE_WORDS: Readonly<Record<string, string>> = {
  [LoopSurveying]: 'lendo as pastas',
  [LoopQualifying]: 'qualificando',
  [LoopAwaitingReview]: 'esperando revisão',
  [LoopWorking]: 'trabalhando',
  [LoopFinished]: 'terminou',
  [LoopStopped]: 'parou',
  [LoopFailed]: 'falhou',
};

const PHASE_WORDS: Readonly<Record<string, string>> = {
  pending: 'pendente',
  qualifying: 'qualificando',
  proposed: 'proposta',
  working: 'trabalhando',
  done: 'pronta',
  unfinished: 'ficou por fazer',
  skipped: 'pulada',
};

export interface LoopFolderView {
  path: string;
  phase: string;
  label: string;
  current: boolean;
  /** The folder's session, to open it from the ruler. */
  sessionId: string | null;
}

/** What the ruler at the top of a loop's window says (D55). */
export interface LoopRulerView {
  id: string;
  /** The goal, else the folder, else the command. */
  title: string;
  state: string;
  tone: 'accent' | 'ok' | 'warn' | 'err' | 'dim';
  /** "pasta 2 de 3 · specs/b", while the run is in a folder. */
  step: string | null;
  elapsed: string;
  folders: LoopFolderView[];
  verdict: string | null;
  error: string | null;
  warnings: string[];
  live: boolean;
}

function verdictText(r: LoopRun): string | null {
  const v = r.verdict;
  if (!v) return null;
  const parts = [`${plural(v.worked, 'pasta trabalhada', 'pastas trabalhadas')}`, `${v.met} de ${plural(v.total, 'critério', 'critérios')}`];
  if (v.unmet?.length) parts.push(`não cumpridos: ${v.unmet.join(', ')}`);
  if (v.unchecked?.length) parts.push(`não conferidos: ${v.unchecked.join(', ')}`);
  if (v.unfinished?.length) parts.push(`ficaram por fazer: ${v.unfinished.join(', ')}`);
  return parts.join(' · ');
}

function toneOf(r: LoopRun): LoopRulerView['tone'] {
  switch (r.state) {
    case LoopFailed:
      return 'err';
    case LoopAwaitingReview:
      return 'warn';
    case LoopStopped:
      return 'dim';
    case LoopFinished: {
      const v = r.verdict;
      const left = !v || v.met < v.total || !!v.unfinished?.length || !!v.unchecked?.length;
      return left ? 'warn' : 'ok';
    }
    default:
      return 'accent';
  }
}

export function loopRuler(r: LoopRun, now: number): LoopRulerView {
  const at = r.specs[r.current];
  const ended = r.finished ? Date.parse(r.finished) : Number.NaN;
  return {
    id: r.id,
    title: r.goal || (r.specs.length === 1 ? (r.specs[0]?.path ?? '') : '') || '/loop',
    state: STATE_WORDS[r.state] ?? r.state,
    tone: toneOf(r),
    step: at && isLive(r) ? `pasta ${r.current + 1} de ${r.specs.length} · ${at.path}` : null,
    elapsed: elapsed(since(r.started, Number.isNaN(ended) ? now : ended)),
    folders: r.specs.map((f, i) => ({
      path: f.path,
      phase: f.phase,
      label: PHASE_WORDS[f.phase] ?? f.phase,
      current: i === r.current && isLive(r),
      sessionId: f.session_id ?? null,
    })),
    verdict: verdictText(r),
    error: r.error || null,
    warnings: r.warnings ?? [],
    live: isLive(r),
  };
}

/** What the "Começar loop" sheet chose. */
export interface LoopChoice {
  /** A folder, or a sentence. */
  target: string;
  protect: readonly string[];
}

/**
 * The argument the sheet sends, written as a person would type it after
 * `/loop`: one grammar, the daemon's, for the field and the sheet (Q6 C).
 */
export function loopArgsOf(c: LoopChoice): { ok: true; args: string } | { ok: false; why: string } {
  const target = c.target.trim();
  if (!target) return { ok: false, why: 'Escolha uma pasta ou escreva um objetivo.' };
  if (target.startsWith('-')) return { ok: false, why: `O loop começa por uma pasta ou um objetivo, não por “${target.split(/\s/)[0]}”.` };
  const globs = c.protect.map((g) => g.trim()).filter(Boolean);
  const spaced = globs.find((g) => /\s/.test(g));
  if (spaced) return { ok: false, why: `Um caminho protegido não tem espaço: “${spaced}”.` };
  return { ok: true, args: [target, ...globs.map((g) => `--protect ${g}`)].join(' ') };
}

/**
 * What the window says on its own about loops that moved (D55): a run that
 * failed, with why — once, when it fails, never for a run that had failed
 * before the window first heard of it.
 */
export function loopNews(before: LoopsState, after: LoopsState): string[] {
  if (!before.loaded) return [];
  const out: string[] = [];
  for (const id of after.order) {
    const was = before.byId[id];
    const is = after.byId[id];
    if (!is || is.state !== LoopFailed || was?.state === LoopFailed) continue;
    out.push(`O loop em ${is.workspace} falhou: ${is.error || 'o daemon não disse por quê'}`);
  }
  return out;
}

/** Why a loop did not start, in the window's words, with the daemon's reason. */
export function startRefusal(code: string, message: string): string {
  if (code === CodeLoopAlreadyRunning) return `Já há um loop vivo neste projeto, e outro não começa enquanto ele roda: ${message}`;
  return `O loop não começou: ${message}`;
}
