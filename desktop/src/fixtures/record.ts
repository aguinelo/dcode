// Helpers for writing a recording by hand: every event is typed by the payload
// its type carries, so a fixture cannot say something the protocol cannot.

import type * as P from '../protocol/generated';
import type { EventKind, PayloadByType, ProtocolEvent } from '../protocol/events';

export interface Recording {
  /** The instant the recording describes; event times are relative to it. */
  recordedAt: string;
  /** The version of the daemon the events came from. */
  daemonVersion: string;
  events: ProtocolEvent[];
}

export function secondsBefore(recordedAt: string, seconds: number): string {
  return new Date(Date.parse(recordedAt) - seconds * 1000).toISOString();
}

export const MINUTE = 60;
export const HOUR = 60 * MINUTE;
export const DAY = 24 * HOUR;

/** Writes one session's events with consecutive seqs, as its log would hold them. */
export class SessionWriter {
  private seq = 0;
  readonly events: ProtocolEvent[] = [];

  constructor(
    readonly id: string,
    private readonly recordedAt: string,
  ) {}

  emit<K extends EventKind>(type: K, secondsAgo: number, payload: PayloadByType[K]): this {
    this.seq += 1;
    const ev = { seq: this.seq, session_id: this.id, type, at: secondsBefore(this.recordedAt, secondsAgo), payload };
    this.events.push(ev as ProtocolEvent);
    return this;
  }

  created(secondsAgo: number, s: Omit<P.Session, 'id' | 'created_at' | 'state' | 'last_seq' | 'done_criteria'>): this {
    return this.emit('session.created', secondsAgo, {
      id: this.id,
      state: 'idle',
      created_at: secondsBefore(this.recordedAt, secondsAgo),
      last_seq: 0,
      first_seq: 1,
      done_criteria: 0,
      ...s,
    });
  }
}

/**
 * The instants of a finished turn, spread over its last `seconds`: the tool
 * calls, the answer and the end, the way a short idle session ends up in the
 * log.
 */
export function finishedTurn(
  w: SessionWriter,
  endedAgo: number,
  turn: {
    id: string;
    ask: string;
    tools: Array<{ id: string; name: string; input: unknown; result: Omit<P.ToolCompleted, 'tool_call_id'> }>;
    answer: string;
    completed: Omit<P.TurnCompleted, 'turn_id'>;
  },
): SessionWriter {
  const start = endedAgo + 60 + turn.tools.length * 20;
  w.emit('turn.started', start, { turn_id: turn.id, text: turn.ask });
  let t = start - 5;
  for (const tool of turn.tools) {
    w.emit('tool.requested', t, { turn_id: turn.id, tool_call_id: tool.id, name: tool.name, input: tool.input });
    w.emit('tool.completed', t - 10, { tool_call_id: tool.id, ...tool.result });
    t -= 20;
  }
  w.emit('message.delta', endedAgo + 5, { turn_id: turn.id, text: turn.answer });
  return w.emit('turn.completed', endedAgo, { turn_id: turn.id, ...turn.completed });
}

/**
 * The recording replayed as if it had just happened: every instant moves by
 * the same amount, so what was 1m35s old when recorded is 1m35s old now, and
 * the live clocks count on from there.
 */
export function asOf(rec: Recording, now: number): ProtocolEvent[] {
  const shift = now - Date.parse(rec.recordedAt);
  const move = (iso: string) => new Date(Date.parse(iso) + shift).toISOString();
  return rec.events.map((ev) => {
    const at = move(ev.at);
    if (ev.type === 'session.created') {
      return { ...ev, at, payload: { ...ev.payload, created_at: move(ev.payload.created_at) } };
    }
    return { ...ev, at };
  });
}
