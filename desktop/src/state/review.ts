// A proposed definition of done, reviewed before any work (D56, Q3 A). Pure:
// the card is drawn from what the daemon measured and announced in
// done.proposed — each criterion's class and what running it did — and never
// from the summary's prose. Editing the criteria stays in the file.

import { CodeLoopNotAwaitingReview, LoopAwaitingReview, SpecProposed, type DoneCriterion, type DoneProposal, type EarlierProposal, type LoopRun } from '../protocol/generated';
import { elapsed, plural } from './format';
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

/** A criterion of an earlier proposal, as the model sent it: nothing was run. */
export interface EarlierCriterion {
  name: string;
  command: string;
  /** What the proposer claimed it would do, in words; null when it claimed nothing. */
  said: string | null;
  why: string | null;
}

/** A proposal the qualifying turn replaced with a later one (D62). */
export interface EarlierCard {
  sequence: number;
  title: string;
  criteria: EarlierCriterion[];
  protected: string[];
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
  /** Every proposal the turn made before the written one, oldest first (D62). */
  earlier: EarlierCard[];
  /** How many earlier proposals there were, in words; null when the turn proposed once. */
  earlierLabel: string | null;
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

const SAID: Readonly<Record<string, string>> = { fail: 'disse que falharia', pass: 'disse que passaria' };

/** The earlier proposals, timed against the written one rather than a clock, so no time zone enters the card. */
function earlierOf(p: DoneProposal): EarlierCard[] {
  const last = Date.parse(p.proposed_at);
  return (p.earlier ?? []).map((e: EarlierProposal) => {
    const gap = Date.parse(e.proposed_at);
    const when = Number.isNaN(last) || Number.isNaN(gap) ? '' : ` · ${elapsed(last - gap)} antes da última`;
    return {
      sequence: e.sequence,
      title: `#${e.sequence} · ${plural(e.criteria.length, 'critério', 'critérios')}${when}`,
      criteria: e.criteria.map((c) => ({ name: c.name, command: c.command, said: (c.expects && SAID[c.expects]) || null, why: c.why || null })),
      protected: e.protected ?? [],
    };
  });
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
          earlier: [],
          earlierLabel: null,
          missing: 'A proposta desta pasta ainda não chegou à janela: abra o arquivo para lê-la.',
        };
      }
      return {
        folder: s.path,
        file,
        sessionId: s.session_id ?? null,
        heading: `${plural(p.criteria.length, 'critério', 'critérios')} · medidos antes do trabalho${p.sequence > 1 ? ` · ${p.sequence}ª proposta do turno` : ''}`,
        criteria: p.criteria.map(rowOf),
        protected: p.protected ?? [],
        noAcceptance: p.no_acceptance ? NO_ACCEPTANCE : null,
        earlier: earlierOf(p),
        earlierLabel: p.earlier?.length ? plural(p.earlier.length, 'proposta anterior', 'propostas anteriores') : null,
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
