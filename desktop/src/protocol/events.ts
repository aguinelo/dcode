// The events the daemon sends, each paired with the payload it carries.
//
// The payload types are generated from internal/protocol. What is written here
// by hand is the pairing itself, which the Go code states in comments and the
// spec rather than in a type — and a unit test holds this list to the
// generated constants, so an event type the protocol adds fails the desktop's
// tests until somebody decides how to show it.

import type * as P from './generated';
import { check, isRecord, opt, req, type AnyShape, type Shape } from './validate';

export interface PayloadByType {
  'session.created': P.Session;
  'session.resumed': P.SessionResumed;
  'session.renamed': P.SessionRenamed;
  'session.mode_changed': P.SessionModeChanged;
  'session.compacted': P.SessionCompacted;
  'session.error': P.Error;
  'session.notice': P.Notice;
  'skill.loaded': P.SkillLoaded;
  'context.band': P.ContextBand;
  'turn.started': P.TurnStarted;
  'turn.steered': P.TurnSteered;
  'turn.completed': P.TurnCompleted;
  'message.delta': P.MessageDelta;
  'message.reasoning': P.MessageReasoning;
  'tool.requested': P.ToolRequested;
  'tool.completed': P.ToolCompleted;
  'tool.approval_required': P.ApprovalRequest;
  'tool.approval_resolved': P.ApprovalResolved;
  progress: P.Progress;
  'plan.updated': P.PlanUpdated;
  'done.proposed': P.DoneProposal;
  // Declared by the protocol and emitted by nothing yet, with no payload type
  // of its own. Read as "signed" and nothing more.
  'done.signed': unknown;
}

export type EventKind = keyof PayloadByType;

export type ProtocolEvent = {
  [K in EventKind]: {
    seq: number;
    session_id: string;
    type: K;
    at: string;
    payload: PayloadByType[K];
  };
}[EventKind];

export type EventOf<K extends EventKind> = Extract<ProtocolEvent, { type: K }>;

const SESSION: Shape<P.Session> = {
  id: req('string'),
  state: req('string'),
  workspace: req('string'),
  model: req('string'),
  family: opt('string'),
  transport: opt('string'),
  base_url: opt('string'),
  branch: opt('string'),
  sandbox_mode: req('string'),
  mode: req('string'),
  created_at: req('string'),
  last_seq: req('number'),
  first_seq: opt('number'),
  context_window: opt('number'),
  done_criteria: req('number'),
};

const USAGE: Shape<P.Usage> = {
  input_tokens: req('number'),
  output_tokens: req('number'),
  cache_read_tokens: opt('number'),
  cache_write_tokens: opt('number'),
  context_tokens: opt('number'),
};

const COMPLETION: Shape<P.Completion> = {
  verification: req('string'),
  met: opt('array', 'string'),
  unmet: opt('array', 'string'),
  unavailable: opt('array', 'string'),
  touched_protected: opt('array', 'string'),
};

const PLAN_ITEM: Shape<P.PlanItem> = {
  id: req('number'),
  text: req('string'),
  status: req('string'),
  blocked: opt('string'),
};

const DONE_CRITERION: Shape<P.DoneCriterion> = {
  name: req('string'),
  command: req('string'),
  exit_code: req('number'),
  expects: opt('string'),
  why: opt('string'),
  class: req('string'),
  exit: req('number'),
  output: opt('string'),
  mismatch: opt('boolean'),
};

const SHAPES: { readonly [K in EventKind]: AnyShape | null } = {
  'session.created': SESSION,
  'session.resumed': {
    source_id: req('string'),
    turns: req('number'),
    started_at: opt('string'),
  } satisfies Shape<P.SessionResumed>,
  'session.renamed': { name: req('string') } satisfies Shape<P.SessionRenamed>,
  'session.mode_changed': {
    previous: opt('string'),
    mode: req('string'),
    sandbox_mode: req('string'),
  } satisfies Shape<P.SessionModeChanged>,
  'session.compacted': {
    from_seq: req('number'),
    to_seq: req('number'),
    messages: opt('number'),
    kept: opt('number'),
  } satisfies Shape<P.SessionCompacted>,
  'session.error': { code: req('string'), message: req('string') } satisfies Shape<P.Error>,
  'session.notice': { code: req('string'), message: req('string') } satisfies Shape<P.Notice>,
  'skill.loaded': { name: req('string'), when_to_use: opt('string') } satisfies Shape<P.SkillLoaded>,
  'context.band': { band: req('number'), fraction: req('number') } satisfies Shape<P.ContextBand>,
  'turn.started': { turn_id: req('string'), text: opt('string') } satisfies Shape<P.TurnStarted>,
  'turn.steered': { turn_id: req('string'), text: req('string') } satisfies Shape<P.TurnSteered>,
  'turn.completed': {
    turn_id: req('string'),
    reason: req('string'),
    usage: opt('object', USAGE),
    completion: opt('object', COMPLETION),
  } satisfies Shape<P.TurnCompleted>,
  'message.delta': { turn_id: req('string'), text: req('string') } satisfies Shape<P.MessageDelta>,
  'message.reasoning': {
    turn_id: req('string'),
    text: req('string'),
  } satisfies Shape<P.MessageReasoning>,
  'tool.requested': {
    turn_id: req('string'),
    tool_call_id: req('string'),
    name: req('string'),
    input: opt('any'),
    typed: opt('boolean'),
  } satisfies Shape<P.ToolRequested>,
  'tool.completed': {
    tool_call_id: req('string'),
    ok: req('boolean'),
    output: req('string'),
    truncated: req('boolean'),
    lines: opt('number'),
    files: opt('number'),
    added: opt('number'),
    removed: opt('number'),
    exit_code: opt('number'),
    has_exit: opt('boolean'),
    duration_ms: opt('number'),
    started_at: opt('string'),
    finished_at: opt('string'),
    diff: opt('string'),
  } satisfies Shape<P.ToolCompleted>,
  'tool.approval_required': {
    approval_id: req('string'),
    turn_id: req('string'),
    tool_call_id: req('string'),
    tool: req('string'),
    command: opt('string'),
    boundary_crossed: req('string'),
    expires_at: req('string'),
    reason: opt('string'),
    rule: opt('string'),
  } satisfies Shape<P.ApprovalRequest>,
  'tool.approval_resolved': {
    approval_id: req('string'),
    decision: req('string'),
  } satisfies Shape<P.ApprovalResolved>,
  progress: {
    turn_id: req('string'),
    tool_call_id: opt('string'),
    name: opt('string'),
    kind: req('string'),
    done: req('number'),
    total: opt('number'),
  } satisfies Shape<P.Progress>,
  'plan.updated': { items: req('array', PLAN_ITEM) } satisfies Shape<P.PlanUpdated>,
  'done.proposed': {
    proposal_id: req('string'),
    turn_id: req('string'),
    round: req('number'),
    criteria: req('array', DONE_CRITERION),
    protected: opt('array', 'string'),
    no_acceptance: opt('boolean'),
    expires_at: req('string'),
  } satisfies Shape<P.DoneProposal>,
  'done.signed': null,
};

/** Every event type this desktop knows how to read. */
export const EVENT_KINDS = Object.keys(SHAPES) as EventKind[];

function isEventKind(t: string): t is EventKind {
  return Object.prototype.hasOwnProperty.call(SHAPES, t);
}

/** An event that could not be read, and why — said, never dropped. */
export interface Problem {
  sessionId: string | null;
  seq: number | null;
  at: string | null;
  type: string | null;
  reason: string;
}

export type Decoded = { ok: true; event: ProtocolEvent } | { ok: false; problem: Problem };

function str(v: unknown): string | null {
  return typeof v === 'string' ? v : null;
}

/** Reads one event as it came off the wire. */
export function decodeEvent(raw: unknown): Decoded {
  if (!isRecord(raw)) {
    return { ok: false, problem: { sessionId: null, seq: null, at: null, type: null, reason: 'o evento não é um objeto' } };
  }
  const sessionId = str(raw.session_id);
  const seq = typeof raw.seq === 'number' && Number.isInteger(raw.seq) && raw.seq > 0 ? raw.seq : null;
  const at = str(raw.at);
  const type = str(raw.type);
  const fail = (reason: string): Decoded => ({ ok: false, problem: { sessionId, seq, at, type, reason } });

  if (!sessionId) return fail('o evento não diz de qual sessão é');
  if (seq === null) return fail('o evento não tem um seq inteiro e positivo');
  if (at === null || Number.isNaN(Date.parse(at))) return fail('o evento não tem um instante legível em at');
  if (type === null) return fail('o evento não diz o tipo');
  if (!isEventKind(type)) return fail(`tipo de evento que esta versão não conhece: ${type}`);

  const shape = SHAPES[type];
  if (shape !== null) {
    const why = check(shape, raw.payload);
    if (why) return fail(`${type}: ${why}`);
  }
  return { ok: true, event: raw as unknown as ProtocolEvent };
}
