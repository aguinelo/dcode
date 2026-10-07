// One session's view state, folded from its protocol events.
//
// Pure: the same events in the same order always produce the same state, which
// is what makes a replay look like having watched the session live. Nothing
// here reads a clock — durations are the view's business, measured from the
// instants the events carry.

import type * as P from '../protocol/generated';
import type { EventOf, ProtocolEvent } from '../protocol/events';

export type ToolStatus = 'arriving' | 'running' | 'ok' | 'failed';

export interface ToolCall {
  id: string;
  name: string;
  input: unknown;
  typed: boolean;
  status: ToolStatus;
  /** When tool.requested (or the first progress of a call still arriving) came. */
  requestedAt: string;
  result: P.ToolCompleted | null;
  progress: { kind: string; done: number; total: number | null } | null;
}

/** Something the flow says that is not a message or a call. */
export type Note =
  | { kind: 'resumed'; sourceId: string; turns: number }
  | { kind: 'notice'; code: string; message: string }
  | { kind: 'mode'; previous: string; mode: string; sandbox: string }
  | { kind: 'compacted'; messages: number; kept: number }
  | { kind: 'skill'; name: string; whenToUse: string }
  | { kind: 'context'; fraction: number }
  | { kind: 'completion'; completion: P.Completion }
  | { kind: 'stopped'; reason: string; rounds: number; maxRounds: number | null }
  | { kind: 'done-proposed'; criteria: number; noAcceptance: boolean }
  | { kind: 'done-signed' }
  | { kind: 'gap'; from: number; to: number }
  | { kind: 'orphan-result'; callId: string }
  | { kind: 'protocol'; reason: string };

interface Stamp {
  seq: number;
  at: string;
}

export type Entry =
  | (Stamp & { kind: 'user'; text: string; steer: boolean })
  | (Stamp & { kind: 'model'; turnId: string; round: number; text: string })
  | (Stamp & { kind: 'reasoning'; turnId: string; round: number; text: string })
  | (Stamp & { kind: 'tool'; call: ToolCall })
  | (Stamp & { kind: 'approval'; request: P.ApprovalRequest; decision: string | null })
  | (Stamp & { kind: 'plan'; items: P.PlanItem[] })
  | (Stamp & { kind: 'note'; note: Note })
  | (Stamp & { kind: 'error'; code: string; message: string });

export interface Counter {
  done: number;
  total: number | null;
}

export interface SessionView {
  id: string;
  /** The session as session.created described it; null until that arrives. */
  info: P.Session | null;
  /** The name a person gave; empty means the title is derived. */
  name: string;
  /** The first thing asked, which the derived title comes from. */
  firstQuestion: string;
  state: string;
  mode: string;
  sandbox: string;
  /** Lines added and removed, summed as the tools reported them. */
  added: number;
  removed: number;
  /** What the assembled context cost at the end of the last turn. */
  contextTokens: number | null;
  /** Turns that completed. */
  turns: number;
  turn: { id: string; startedAt: string } | null;
  rounds: Counter | null;
  inFlight: Counter | null;
  pendingApprovalId: string | null;
  lastSeq: number;
  lastAt: string | null;
  entries: Entry[];
}

export function emptySession(id: string): SessionView {
  return {
    id,
    info: null,
    name: '',
    firstQuestion: '',
    state: 'idle',
    mode: '',
    sandbox: '',
    added: 0,
    removed: 0,
    contextTokens: null,
    turns: 0,
    turn: null,
    rounds: null,
    inFlight: null,
    pendingApprovalId: null,
    lastSeq: 0,
    lastAt: null,
    entries: [],
  };
}

/** A note about the stream itself, placed in the session it concerns. */
export function withNote(view: SessionView, seq: number, at: string, note: Note): SessionView {
  return { ...view, entries: [...view.entries, { kind: 'note', seq, at, note }] };
}

function counter(done: number, total: number | undefined): Counter {
  return { done, total: total && total > 0 ? total : null };
}

function updateCall(entries: Entry[], callId: string, update: (c: ToolCall) => ToolCall): Entry[] | null {
  for (let i = entries.length - 1; i >= 0; i--) {
    const e = entries[i];
    if (e?.kind === 'tool' && e.call.id === callId) {
      const next = entries.slice();
      next[i] = { ...e, call: update(e.call) };
      return next;
    }
  }
  return null;
}

function appendText(
  entries: Entry[],
  kind: 'model' | 'reasoning',
  round: number,
  ev: { seq: number; at: string; payload: { turn_id: string; text: string } },
): Entry[] {
  const last = entries[entries.length - 1];
  // Text streams in fragments; it reads as one message only while it flows
  // into the entry already open for the same turn and the same round. A turn
  // the done check sends round again speaks anew, with nothing between the
  // two messages but the daemon's report that the round moved.
  if (last && last.kind === kind && last.turnId === ev.payload.turn_id && last.round === round) {
    return [...entries.slice(0, -1), { ...last, text: last.text + ev.payload.text }];
  }
  return [...entries, { kind, seq: ev.seq, at: ev.at, turnId: ev.payload.turn_id, round, text: ev.payload.text }];
}

function onProgress(view: SessionView, ev: EventOf<'progress'>): SessionView {
  const d = ev.payload;
  if (d.tool_call_id) {
    const progress = { kind: d.kind, done: d.done, total: d.total && d.total > 0 ? d.total : null };
    const updated = updateCall(view.entries, d.tool_call_id, (c) => ({ ...c, progress }));
    if (updated) return { ...view, entries: updated };
    // A call still arriving has no entry yet: tool.requested only comes once
    // the model has finished sending it. The report names the tool for exactly
    // this, so a line can appear the moment the call starts.
    if (d.name) {
      const call: ToolCall = {
        id: d.tool_call_id,
        name: d.name,
        input: null,
        typed: false,
        status: 'arriving',
        requestedAt: ev.at,
        result: null,
        progress,
      };
      return { ...view, entries: [...view.entries, { kind: 'tool', seq: ev.seq, at: ev.at, call }] };
    }
    return view;
  }
  switch (d.kind) {
    case 'rounds':
      return { ...view, rounds: counter(d.done, d.total) };
    case 'in_flight':
      return { ...view, inFlight: counter(d.done, d.total) };
    default:
      return view;
  }
}

function onToolRequested(view: SessionView, ev: EventOf<'tool.requested'>): SessionView {
  const d = ev.payload;
  // The call announced itself while it was arriving, so its line is already
  // there: fill it in rather than drawing a second one.
  const arriving = view.entries.findIndex(
    (e) => e.kind === 'tool' && e.call.id === d.tool_call_id && e.call.status === 'arriving',
  );
  if (arriving >= 0) {
    const filled = updateCall(view.entries, d.tool_call_id, (c) => ({
      ...c,
      name: d.name,
      input: d.input ?? null,
      typed: !!d.typed,
      status: 'running',
      progress: null,
    }));
    if (filled) return { ...view, entries: filled };
  }
  const call: ToolCall = {
    id: d.tool_call_id,
    name: d.name,
    input: d.input ?? null,
    typed: !!d.typed,
    status: 'running',
    requestedAt: ev.at,
    result: null,
    progress: null,
  };
  return { ...view, entries: [...view.entries, { kind: 'tool', seq: ev.seq, at: ev.at, call }] };
}

function onToolCompleted(view: SessionView, ev: EventOf<'tool.completed'>): SessionView {
  const d = ev.payload;
  const added = d.added ?? 0;
  const removed = d.removed ?? 0;
  const updated = updateCall(view.entries, d.tool_call_id, (c) => ({
    ...c,
    status: d.ok ? 'ok' : 'failed',
    result: d,
  }));
  const next = { ...view, added: view.added + added, removed: view.removed + removed };
  if (updated) return { ...next, entries: updated };
  // A result for a call nobody announced is the log disagreeing with itself.
  return withNote(next, ev.seq, ev.at, { kind: 'orphan-result', callId: d.tool_call_id });
}

function onApprovalResolved(view: SessionView, ev: EventOf<'tool.approval_resolved'>): SessionView {
  const d = ev.payload;
  // The answer lands on its own question, matched by id: two crossings can be
  // outstanding, and an answer on the wrong card is a decision nobody made.
  const entries = view.entries.map((e) =>
    e.kind === 'approval' && e.request.approval_id === d.approval_id ? { ...e, decision: d.decision } : e,
  );
  return {
    ...view,
    entries,
    pendingApprovalId: view.pendingApprovalId === d.approval_id ? null : view.pendingApprovalId,
    state: view.state === 'blocked' ? 'running' : view.state,
  };
}

function onPlan(view: SessionView, ev: EventOf<'plan.updated'>): SessionView {
  const items = ev.payload.items ?? [];
  // One block, updated where it first appeared: every revision appended would
  // stack the same plan down the screen.
  const at = view.entries.findIndex((e) => e.kind === 'plan');
  if (at < 0) return { ...view, entries: [...view.entries, { kind: 'plan', seq: ev.seq, at: ev.at, items }] };
  const entries = view.entries.slice();
  const current = entries[at];
  if (current?.kind === 'plan') entries[at] = { ...current, items };
  return { ...view, entries };
}

// The stop reasons that end a turn abruptly and say nothing else about it.
// `error` already arrived as session.error; `done` is the answer itself.
const ABRUPT = new Set(['max_iterations', 'repeat_loop', 'max_tokens', 'interrupted']);

function onTurnCompleted(view: SessionView, ev: EventOf<'turn.completed'>): SessionView {
  const d = ev.payload;
  let next: SessionView = {
    ...view,
    state: 'idle',
    turn: null,
    turns: view.turns + 1,
    contextTokens: d.usage?.context_tokens ? d.usage.context_tokens : view.contextTokens,
  };
  if (d.completion && d.completion.verification !== 'clean') {
    next = withNote(next, ev.seq, ev.at, { kind: 'completion', completion: d.completion });
  } else if (ABRUPT.has(d.reason)) {
    next = withNote(next, ev.seq, ev.at, {
      kind: 'stopped',
      reason: d.reason,
      rounds: view.rounds?.done ?? 0,
      maxRounds: view.rounds?.total ?? null,
    });
  }
  return next;
}

function applyKnown(view: SessionView, ev: ProtocolEvent): SessionView {
  switch (ev.type) {
    case 'session.created': {
      const s = ev.payload;
      return { ...view, info: s, state: s.state, mode: s.mode, sandbox: s.sandbox_mode };
    }
    case 'session.resumed':
      return withNote(view, ev.seq, ev.at, {
        kind: 'resumed',
        sourceId: ev.payload.source_id,
        turns: ev.payload.turns,
      });
    case 'session.renamed':
      return { ...view, name: ev.payload.name };
    case 'session.mode_changed': {
      const d = ev.payload;
      const next = { ...view, mode: d.mode || view.mode, sandbox: d.sandbox_mode || view.sandbox };
      // The first announce has nothing to come from; only a switch is news.
      if (!d.previous) return next;
      return withNote(next, ev.seq, ev.at, { kind: 'mode', previous: d.previous, mode: d.mode, sandbox: d.sandbox_mode });
    }
    case 'session.compacted':
      return withNote(view, ev.seq, ev.at, {
        kind: 'compacted',
        messages: ev.payload.messages ?? 0,
        kept: ev.payload.kept ?? 0,
      });
    case 'session.error':
      return {
        ...view,
        entries: [...view.entries, { kind: 'error', seq: ev.seq, at: ev.at, code: ev.payload.code, message: ev.payload.message }],
      };
    case 'session.notice':
      // What the session says as it opens: a warning, not a failure, so a note
      // in the flow rather than an error entry.
      return withNote(view, ev.seq, ev.at, { kind: 'notice', code: ev.payload.code, message: ev.payload.message });
    case 'skill.loaded':
      return withNote(view, ev.seq, ev.at, {
        kind: 'skill',
        name: ev.payload.name,
        whenToUse: ev.payload.when_to_use ?? '',
      });
    case 'context.band':
      if (ev.payload.fraction <= 0) return view;
      return withNote(view, ev.seq, ev.at, { kind: 'context', fraction: ev.payload.fraction });
    case 'turn.started': {
      const text = ev.payload.text ?? '';
      const entries: Entry[] = text
        ? [...view.entries, { kind: 'user', seq: ev.seq, at: ev.at, text, steer: false }]
        : view.entries;
      return {
        ...view,
        entries,
        firstQuestion: view.firstQuestion || text,
        state: 'running',
        turn: { id: ev.payload.turn_id, startedAt: ev.at },
        // The counters belong to the turn that is starting.
        rounds: null,
        inFlight: null,
      };
    }
    case 'turn.steered':
      if (!ev.payload.text) return view;
      return {
        ...view,
        entries: [...view.entries, { kind: 'user', seq: ev.seq, at: ev.at, text: ev.payload.text, steer: true }],
      };
    case 'turn.completed':
      return onTurnCompleted(view, ev);
    case 'message.delta':
      return { ...view, entries: appendText(view.entries, 'model', view.rounds?.done ?? 0, ev) };
    case 'message.reasoning':
      return { ...view, entries: appendText(view.entries, 'reasoning', view.rounds?.done ?? 0, ev) };
    case 'tool.requested':
      return onToolRequested(view, ev);
    case 'tool.completed':
      return onToolCompleted(view, ev);
    case 'tool.approval_required':
      return {
        ...view,
        state: 'blocked',
        pendingApprovalId: ev.payload.approval_id,
        entries: [...view.entries, { kind: 'approval', seq: ev.seq, at: ev.at, request: ev.payload, decision: null }],
      };
    case 'tool.approval_resolved':
      return onApprovalResolved(view, ev);
    case 'progress':
      return onProgress(view, ev);
    case 'plan.updated':
      return onPlan(view, ev);
    case 'done.proposed':
      return withNote(view, ev.seq, ev.at, {
        kind: 'done-proposed',
        criteria: (ev.payload.criteria ?? []).length,
        noAcceptance: !!ev.payload.no_acceptance,
      });
    case 'done.signed':
      return withNote(view, ev.seq, ev.at, { kind: 'done-signed' });
  }
}

/**
 * Folds one event into its session.
 *
 * A seq at or below the last one is a replay overlap and changes nothing —
 * reconnecting from an earlier point is how a client catches up, and it must
 * be idempotent. A seq that skips ahead is the log gapping, which it promises
 * never to do, so the gap is said in the flow rather than drawn over.
 */
export function applyEvent(view: SessionView, ev: ProtocolEvent): SessionView {
  if (ev.seq <= view.lastSeq) return view;
  let next = view;
  if (view.lastSeq > 0 && ev.seq > view.lastSeq + 1) {
    next = withNote(next, ev.seq, ev.at, { kind: 'gap', from: view.lastSeq + 1, to: ev.seq - 1 });
  }
  next = applyKnown(next, ev);
  return { ...next, lastSeq: ev.seq, lastAt: ev.at };
}
