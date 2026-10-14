// A proposed definition of done, reviewed before any work (D56, Q3 A). Pure:
// the card is drawn from what the daemon measured and announced in
// done.proposed — each criterion's class and what running it did — and never
// from the summary's prose. Editing the criteria stays in the file.

import { CodeLoopNotAwaitingReview, LoopAwaitingReview, SpecProposed, type DoneCriterion, type LoopRun } from '../protocol/generated';
import { plural } from './format';
import type { SessionView } from './session';

export interface CriterionRow {
  name: string;
  command: string;
  /** The daemon's class: acceptance, regression or broken. */
  klass: string;
  label: string;
  /** What running it before any work did. */
  ran: string;
  passed: boolean;
  why: string | null;
  output: string | null;
  /** Where the eye should land: the proposer disagreeing with the run, or nothing to run. */
  warning: string | null;
}

export interface ProposalCard {
  folder: string;
  /** The done.toml the daemon wrote, as it reported it. */
  file: string;
  sessionId: string | null;
  heading: string;
  criteria: CriterionRow[];
  protected: string[];
  noAcceptance: string | null;
  /** Why there is no card to draw for this folder yet. */
  missing: string | null;
}

export interface ReviewView {
  loopId: string;
  proposals: ProposalCard[];
}

const LABELS: Readonly<Record<string, string>> = { acceptance: 'aceitação', regression: 'regressão', broken: 'quebrado' };

function warningOf(c: DoneCriterion): string | null {
  if (c.class === 'broken') return 'Não havia o que rodar: este critério não testemunha nada e ficaria vermelho para sempre.';
  if (!c.mismatch) return null;
  return c.class === 'acceptance'
    ? 'O proponente disse que passaria, e falhou: confira o porquê.'
    : 'O proponente disse que falharia, e passou: confira o porquê.';
}

function rowOf(c: DoneCriterion): CriterionRow {
  return {
    name: c.name,
    command: c.command,
    klass: c.class,
    label: LABELS[c.class] ?? c.class,
    ran: `exit ${c.exit}`,
    passed: c.exit === c.exit_code,
    why: c.why || null,
    output: c.output || null,
    warning: warningOf(c),
  };
}

export const NO_ACCEPTANCE =
  'Nenhum critério está vermelho: o conjunto diz pronto sem que nada mude. Num refactor pode ser o certo; fora dele, é defeito.';

/**
 * The card for a run waiting for a review, one proposal per folder it wrote —
 * a goal over a backlog qualifies them all in one pass. Null when the run is
 * not waiting: nothing is reviewed after the work started.
 */
export function reviewOf(run: LoopRun, sessions: Readonly<Record<string, SessionView | undefined>>): ReviewView | null {
  if (run.state !== LoopAwaitingReview) return null;
  const proposals = run.specs
    .filter((s) => s.phase === SpecProposed && s.proposal)
    .map((s): ProposalCard => {
      const file = s.proposal ?? '';
      const said = s.session_id ? sessions[s.session_id]?.proposal : null;
      const p = said && said.path === file ? said : null;
      if (!p) {
        return {
          folder: s.path,
          file,
          sessionId: s.session_id ?? null,
          heading: 'proposta escrita',
          criteria: [],
          protected: [],
          noAcceptance: null,
          missing: 'A proposta desta pasta ainda não chegou à janela: abra o arquivo para lê-la.',
        };
      }
      return {
        folder: s.path,
        file,
        sessionId: s.session_id ?? null,
        heading: `${plural(p.criteria.length, 'critério', 'critérios')} · medidos antes do trabalho`,
        criteria: p.criteria.map(rowOf),
        protected: p.protected ?? [],
        noAcceptance: p.no_acceptance ? NO_ACCEPTANCE : null,
        missing: null,
      };
    });
  return { loopId: run.id, proposals };
}

/** Why the work did not start, with the daemon's reason: the first answer wins. */
export function continueRefusal(code: string, message: string): string {
  if (code === CodeLoopNotAwaitingReview) return `O trabalho já começou, ou o loop saiu da revisão — outra janela ou a TUI respondeu antes: ${message}`;
  return `O trabalho não começou: ${message}`;
}
