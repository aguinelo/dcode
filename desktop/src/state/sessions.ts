// Every session the window knows, folded from the events of all of them.

import { decodeEvent, type Decoded, type Problem } from '../protocol/events';
import type { AgentList } from './agents';
import { applyEvent, emptySession, withNote, type SessionView } from './session';

export interface SessionsState {
  byId: Readonly<Record<string, SessionView>>;
  /** Session ids in the order their first event arrived. */
  order: readonly string[];
  /** Events that could not be read and name no session to show them in. */
  problems: readonly Problem[];
}

export const emptySessions: SessionsState = { byId: {}, order: [], problems: [] };

function sessionOf(state: SessionsState, id: string): { state: SessionsState; view: SessionView } {
  const existing = state.byId[id];
  if (existing) return { state, view: existing };
  return { state: { ...state, order: [...state.order, id] }, view: emptySession(id) };
}

function put(state: SessionsState, view: SessionView): SessionsState {
  return { ...state, byId: { ...state.byId, [view.id]: view } };
}

/**
 * Folds one decoded event. An event that could not be read is said where it
 * belongs — in its session's flow when it names one — and never dropped.
 */
export function applyDecoded(state: SessionsState, decoded: Decoded): SessionsState {
  if (decoded.ok) {
    const { state: withSession, view } = sessionOf(state, decoded.event.session_id);
    return put(withSession, applyEvent(view, decoded.event));
  }
  const p = decoded.problem;
  if (!p.sessionId) return { ...state, problems: [...state.problems, p] };
  const { state: withSession, view } = sessionOf(state, p.sessionId);
  const at = p.at ?? view.lastAt ?? new Date(0).toISOString();
  return put(withSession, withNote(view, p.seq ?? view.lastSeq, at, { kind: 'protocol', reason: p.reason }));
}

/**
 * Changes one followed session's agent list with what the daemon answered
 * outside its stream — /children, /questions, an answer (D66). A session the
 * window has no events of yet is left alone: there is no view to change.
 */
export function withAgentsOf(state: SessionsState, id: string, change: (agents: AgentList) => AgentList): SessionsState {
  const view = state.byId[id];
  return view ? put(state, { ...view, agents: change(view.agents) }) : state;
}

/** Decodes and folds a batch of events as they came off the wire. */
export function applyRaw(state: SessionsState, raw: readonly unknown[]): SessionsState {
  return raw.reduce<SessionsState>((s, r) => applyDecoded(s, decodeEvent(r)), state);
}
