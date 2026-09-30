import type { EventKind, PayloadByType, ProtocolEvent } from '../../src/protocol/events';
import { applyEvent, emptySession, type SessionView } from '../../src/state/session';

/** A little log for one session: each call is the next event, one second later. */
export function log(sessionId = 's-1', start = '2026-09-28T17:00:00.000Z') {
  let seq = 0;
  const t0 = Date.parse(start);
  const events: ProtocolEvent[] = [];
  const api = {
    events,
    add<K extends EventKind>(type: K, payload: PayloadByType[K]) {
      seq += 1;
      events.push({ seq, session_id: sessionId, type, at: new Date(t0 + seq * 1000).toISOString(), payload } as ProtocolEvent);
      return api;
    },
    fold(from: SessionView = emptySession(sessionId)): SessionView {
      return events.reduce(applyEvent, from);
    },
  };
  return api;
}

export const created = {
  id: 's-1',
  state: 'idle',
  workspace: '/w/dcode',
  model: 'MiniMax-M3',
  branch: 'feat/x',
  sandbox_mode: 'workspace-write',
  mode: 'assist',
  created_at: '2026-09-28T17:00:00.000Z',
  last_seq: 0,
  context_window: 200000,
  done_criteria: 0,
};

export function done(callId: string, extra: Partial<PayloadByType['tool.completed']> = {}): PayloadByType['tool.completed'] {
  return { tool_call_id: callId, ok: true, output: '', truncated: false, ...extra };
}
