// The orchestrator's plan with its agents (D68): each step with the children
// whose delegate item named it (Child.plan_step), and each child on the
// pipeline a child moves through — dispatched, working, review, gate, PR,
// merged — at the phase it is in, in the tone of D52 and D61. Pure, and only
// the wire (D8): the phase is read from the child's state, its review and its
// gate, never from text.

import type * as P from '../protocol/generated';

/** The six phases of the pipeline, in order. */
export const PHASES = [
  { key: 'dispatched', word: 'despachado' },
  { key: 'working', word: 'trabalhando' },
  { key: 'review', word: 'revisão' },
  { key: 'gate', word: 'gate' },
  { key: 'pr', word: 'PR' },
  { key: 'merged', word: 'merged' },
] as const;

export interface PhaseView {
  /** The index in PHASES of the phase the child is in, or stopped at. */
  at: number;
  /** accent at work, warn waiting for you or returned, err failed, ok merged, dim dispatched or closed. */
  tone: string;
  /** The state in a word or two, beside the pipeline. */
  word: string;
}

const WAITING: Record<string, string> = { approval: 'pede aprovação', question: 'tem uma pergunta', merge: 'espera o merge' };

/** How far a child that left the pipeline had got: the PR, the gate, the review, or the work. */
function reached(c: P.Child): number {
  if (c.pr) return 4;
  if (c.gate || c.review) return 3;
  return 1;
}

/** Where a child is on the pipeline, and in which tone. A state the app does not know is said in red. */
export function childPhase(c: P.Child): PhaseView {
  switch (c.state) {
    case 'dispatched':
      return { at: 0, tone: 'dim', word: 'despachado' };
    case 'working':
      return c.returned ? { at: 1, tone: 'warn', word: `devolvido ${c.returned}×` } : { at: 1, tone: 'accent', word: 'trabalhando' };
    case 'awaiting_person':
      return { at: 1, tone: 'warn', word: c.waiting ? (WAITING[c.waiting] ?? `espera: ${c.waiting}`) : 'esperando você' };
    case 'in_review':
      if (!c.review) return { at: 2, tone: 'accent', word: 'revisando' };
      return { at: 3, tone: 'accent', word: c.gate ? `gate ${c.gate.result}` : 'medindo o gate' };
    case 'pr':
      return c.waiting ? { at: 4, tone: 'warn', word: WAITING[c.waiting] ?? `espera: ${c.waiting}` } : { at: 4, tone: 'accent', word: 'PR' };
    case 'merged':
      return { at: 5, tone: 'ok', word: 'merged' };
    case 'failed':
      return { at: reached(c), tone: 'err', word: c.reason ? `falhou: ${c.reason}` : 'falhou' };
    case 'closed':
      return { at: reached(c), tone: 'dim', word: 'fechado' };
    default:
      return { at: 0, tone: 'err', word: `estado "${c.state}" desconhecido` };
  }
}

export interface StepAgents {
  /** The children of each plan step, by item id, in the order they started. */
  byStep: Record<number, P.Child[]>;
  /** The children that name no step of this plan: still shown, outside it. */
  outside: P.Child[];
}

/** The children under the step each names; one that names none, or a step the plan does not have, is outside. */
export function stepAgents(items: readonly P.PlanItem[], children: readonly P.Child[]): StepAgents {
  const ids = new Set(items.map((i) => i.id));
  const byStep: Record<number, P.Child[]> = {};
  const outside: P.Child[] = [];
  for (const c of children) {
    const step = c.plan_step ?? 0;
    if (step > 0 && ids.has(step)) (byStep[step] ??= []).push(c);
    else outside.push(c);
  }
  return { byStep, outside };
}
