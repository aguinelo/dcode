// The messages a conversation holds while a turn runs (D47), as the TUI's
// queue does: kept in order, sent one at a time once the conversation is idle
// again. ↵ during a turn still steers it (D23); queuing is ⌥↵.
//
// Pure: what the conversation is doing goes in, whether to send comes out.
// The window does the sending and reports back with `sent` or `failed`.

export type QueuePhase =
  /** Waiting for the conversation to be idle. */
  | 'waiting'
  /** The oldest is on its way; nothing else goes until the daemon answers. */
  | 'sending'
  /** The daemon took it: the next waits for that turn to start and end. */
  | 'sent'
  /** A send failed: nothing goes until the person resumes it. */
  | 'halted';

export interface Queue {
  items: readonly string[];
  phase: QueuePhase;
  /** Turns completed when the last send went, to tell its turn ended. */
  turnsAtSend: number;
  /** Why it halted; empty otherwise. */
  why: string;
}

export type Queues = Readonly<Record<string, Queue>>;

export const noQueues: Queues = {};

/** What the queue needs to know of the conversation. */
export interface Progress {
  state: string;
  turns: number;
}

function put(qs: Queues, id: string, q: Queue | null): Queues {
  const next = { ...qs };
  if (q && (q.items.length > 0 || q.phase === 'sending')) next[id] = q;
  else delete next[id];
  return next;
}

export function enqueue(qs: Queues, id: string, text: string): Queues {
  const q = qs[id] ?? { items: [], phase: 'waiting' as const, turnsAtSend: 0, why: '' };
  return put(qs, id, { ...q, items: [...q.items, text] });
}

/** ^X: the last queued goes. */
export function dropLast(qs: Queues, id: string): Queues {
  const q = qs[id];
  return q ? dropAt(qs, id, q.items.length - 1) : qs;
}

/** ×: the one it is on goes. The one on its way cannot be called back. */
export function dropAt(qs: Queues, id: string, index: number): Queues {
  const q = qs[id];
  if (!q || index < 0 || index >= q.items.length || (q.phase === 'sending' && index === 0)) return qs;
  return put(qs, id, { ...q, items: q.items.filter((_, i) => i !== index) });
}

export function resume(qs: Queues, id: string): Queues {
  const q = qs[id];
  return q && q.phase === 'halted' ? put(qs, id, { ...q, phase: 'waiting', why: '' }) : qs;
}

/** The daemon took the oldest. */
export function sent(qs: Queues, id: string): Queues {
  const q = qs[id];
  return q ? put(qs, id, { ...q, items: q.items.slice(1), phase: 'sent' }) : qs;
}

/** The oldest was refused: it stays, and the queue stops, saying why. */
export function failed(qs: Queues, id: string, why: string): Queues {
  const q = qs[id];
  return q ? put(qs, id, { ...q, phase: 'halted', why }) : qs;
}

/**
 * One look at a conversation: a sent message's turn that started (or already
 * ended) frees the next, and an idle conversation takes the oldest.
 */
export function step(qs: Queues, id: string, p: Progress): { queues: Queues; send: string | null } {
  let q = qs[id];
  if (!q) return { queues: qs, send: null };
  if (q.phase === 'sent' && (p.state !== 'idle' || p.turns > q.turnsAtSend)) q = { ...q, phase: 'waiting' };
  if (q.phase === 'waiting' && p.state === 'idle' && q.items.length > 0) {
    const next = { ...q, phase: 'sending' as const, turnsAtSend: p.turns };
    return { queues: put(qs, id, next), send: next.items[0] ?? null };
  }
  return { queues: q === qs[id] ? qs : put(qs, id, q), send: null };
}

/** One look at every queue; the same object back when nothing moved. A conversation not held is left as it is. */
export function stepAll(qs: Queues, progressOf: (id: string) => Progress | undefined): Queues {
  let next = qs;
  for (const id of Object.keys(qs)) {
    const p = progressOf(id);
    if (p) next = step(next, id, p).queues;
  }
  return next;
}
