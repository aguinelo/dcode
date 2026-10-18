// The orchestrator's events, read at the boundary like every other: the shapes
// of the payloads internal/protocol/orchestrate.go declares. The live agent
// list folds them (D64), and an event the daemon can send is an event this
// desktop reads and checks, never one it lets through unread. The answers of
// the project routes are read in projects.ts, with these shapes.

import type * as P from './generated';
import { opt, req, type Shape } from './validate';

export const USAGE: Shape<P.Usage> = {
  input_tokens: req('number'),
  output_tokens: req('number'),
  cache_read_tokens: opt('number'),
  cache_write_tokens: opt('number'),
  context_tokens: opt('number'),
};

const RETURN: Shape<P.Return> = {
  why: req('string'),
  phase: opt('string'),
  paths: opt('array', 'string'),
};

const FINDING: Shape<P.Finding> = {
  path: req('string'),
  line: opt('number'),
  what: req('string'),
  why: req('string'),
};

const VERDICT: Shape<P.Verdict> = {
  session_id: req('string'),
  commit: req('string'),
  verdict: req('string'),
  findings: opt('array', FINDING),
};

const GATE_RESULT: Shape<P.GateResult> = {
  commit: req('string'),
  tree: req('string'),
  base: req('string'),
  command: req('string'),
  exit_code: req('number'),
  duration_ms: req('number'),
  result: req('string'),
  output_tail: opt('string'),
  output_path: opt('string'),
  sandbox: opt('string'),
  sandbox_mode: opt('string'),
  at: req('string'),
};

const CHILD_PR: Shape<P.ChildPR> = {
  number: opt('number'),
  url: opt('string'),
  ci: opt('string'),
};

export const CHILD: Shape<P.Child> = {
  id: req('string'),
  parent_id: req('string'),
  title: req('string'),
  profile: req('string'),
  model: req('string'),
  state: req('string'),
  reason: opt('string'),
  waiting: opt('string'),
  owns: req('array', 'string'),
  branch: req('string'),
  worktree: req('string'),
  base: req('string'),
  head: opt('string'),
  returned: opt('number'),
  return: opt('object', RETURN),
  review: opt('object', VERDICT),
  gate: opt('object', GATE_RESULT),
  pr: opt('object', CHILD_PR),
  usage: req('object', USAGE),
  started: req('string'),
  ended: opt('string'),
};

const QUESTION_OPTION: Shape<P.QuestionOption> = {
  id: req('string'),
  label: req('string'),
};

export const QUESTION: Shape<P.Question> = {
  id: req('string'),
  kind: req('string'),
  text: req('string'),
  recommended: req('string'),
  why: req('string'),
  child_id: opt('string'),
  options: req('array', QUESTION_OPTION),
  sensitive: opt('array', 'string'),
  asked: req('string'),
  answer: opt('string'),
  note: opt('string'),
  answered: opt('string'),
};

const MAIN_HEALTH: Shape<P.MainHealth> = {
  commit: req('string'),
  health: req('string'),
  checked: req('string'),
};

const CHILD_CHANGED: Shape<P.ChildChanged> = { child: req('object', CHILD) };

/** The payload shape of each orchestrator event. */
export const ORCHESTRATE_SHAPES = {
  'child.started': CHILD_CHANGED,
  'child.state': CHILD_CHANGED,
  'child.completed': {
    child_id: req('string'),
    turn_id: req('string'),
    reason: req('string'),
    conclusion: req('string'),
    truncated: opt('boolean'),
    wrote: opt('array', 'string'),
    unwritten: opt('array', 'string'),
    usage: req('object', USAGE),
  } satisfies Shape<P.ChildCompleted>,
  'review.verdict': { child_id: req('string'), ...VERDICT } satisfies Shape<P.ReviewVerdict>,
  'gate.measured': { child_id: req('string'), ...GATE_RESULT } satisfies Shape<P.GateMeasured>,
  'question.asked': QUESTION,
  'question.answered': QUESTION,
  'merge.completed': {
    child_id: req('string'),
    method: req('string'),
    pr: opt('number'),
    merged: req('string'),
    measured: req('string'),
  } satisfies Shape<P.MergeCompleted>,
  'merge.refused': {
    child_id: req('string'),
    code: req('string'),
    reason: req('string'),
  } satisfies Shape<P.MergeRefused>,
  'main.health': MAIN_HEALTH,
} as const;

/** The orchestrator's events, each paired with the payload it carries. */
export interface OrchestratePayloadByType {
  'child.started': P.ChildChanged;
  'child.state': P.ChildChanged;
  'child.completed': P.ChildCompleted;
  'review.verdict': P.ReviewVerdict;
  'gate.measured': P.GateMeasured;
  'question.asked': P.Question;
  'question.answered': P.Question;
  'merge.completed': P.MergeCompleted;
  'merge.refused': P.MergeRefused;
  'main.health': P.MainHealth;
}
