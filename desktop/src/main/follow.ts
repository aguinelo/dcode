import { CodeEventsExpired, CodeSessionNotFound } from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { Refusal, SessionEvents, StreamEnd } from '../shared/api';
import { sessionPath } from './requests';
import { decodeData, type SseFrame } from './sse';
import { openStream, type Stream, type StreamEnded } from './wire';

// The sessions the window follows, each on its own event stream from the
// daemon. A stream that drops while the daemon still answers is opened again
// from the event after the last one seen, so a session is read whole without
// reading anything twice; one the daemon refuses ends, and says why.

type Timer = ReturnType<typeof setTimeout>;

/** Waits that grow while something keeps failing, and start over once it works. */
export class Backoff {
  private wait: number;

  constructor(
    private readonly fromMs: number,
    private readonly upToMs: number,
  ) {
    this.wait = fromMs;
  }

  next(): number {
    const w = this.wait;
    this.wait = Math.min(this.wait * 2, this.upToMs);
    return w;
  }

  reset(): void {
    this.wait = this.fromMs;
  }
}

interface Followed {
  readonly id: string;
  /** The last seq read; the stream resumes from the one after. */
  lastSeq: number;
  stream: Stream | null;
  retry: Timer | null;
  readonly backoff: Backoff;
}

export interface FollowDeps {
  /** The daemon's socket, once one is known. */
  socket(): string | null;
  retryFromMs: number;
  retryUpToMs: number;
  events(batch: SessionEvents): void;
  ended(end: StreamEnd): void;
  /** A stream dropped: asks /health, and resolves true when the daemon still answers. */
  dropped(): Promise<boolean>;
}

/** The seq of a frame: its `id:`, else the event's own `seq`. */
function seqOf(frame: SseFrame, event: unknown): number | null {
  const id = frame.id === undefined ? NaN : Number(frame.id);
  if (Number.isSafeInteger(id) && id > 0) return id;
  if (isRecord(event) && typeof event.seq === 'number' && Number.isSafeInteger(event.seq)) return event.seq;
  return null;
}

/** Why a session's stream ended for good, with what the daemon said. */
export function endReason(sessionId: string, from: number, refusal: Refusal): string {
  switch (refusal.code) {
    case CodeSessionNotFound:
      return `O daemon não tem mais a sessão ${sessionId}: ${refusal.message}`;
    case CodeEventsExpired:
      return `O daemon não guarda mais os eventos da sessão ${sessionId} a partir do ${from}: ${refusal.message}`;
    default:
      return `O daemon recusou o fluxo da sessão ${sessionId} (${refusal.code}): ${refusal.message}`;
  }
}

export class Followers {
  private readonly followed = new Map<string, Followed>();

  constructor(private readonly d: FollowDeps) {}

  /** Follows a session from its first event; one already followed starts over. */
  follow(id: string): void {
    this.unfollow(id);
    const f: Followed = { id, lastSeq: 0, stream: null, retry: null, backoff: new Backoff(this.d.retryFromMs, this.d.retryUpToMs) };
    this.followed.set(id, f);
    this.open(f);
  }

  unfollow(id: string): void {
    const f = this.followed.get(id);
    if (!f) return;
    this.followed.delete(id);
    this.halt(f);
  }

  /** The daemon stopped answering: every stream closes, and each keeps where it was. */
  suspendAll(): void {
    for (const f of this.followed.values()) this.halt(f);
  }

  /** The daemon answers again: every followed session resumes from the event after its last. */
  resumeAll(): void {
    for (const f of this.followed.values()) {
      this.halt(f);
      this.open(f);
    }
  }

  /** Forgets every session. */
  closeAll(): void {
    this.suspendAll();
    this.followed.clear();
  }

  /** What is followed, and the last seq of each: for the logs and the tests. */
  positions(): Record<string, number> {
    return Object.fromEntries([...this.followed.values()].map((f) => [f.id, f.lastSeq]));
  }

  private current(f: Followed): boolean {
    return this.followed.get(f.id) === f;
  }

  private halt(f: Followed): void {
    f.stream?.close();
    f.stream = null;
    if (f.retry) clearTimeout(f.retry);
    f.retry = null;
  }

  private open(f: Followed): void {
    const socket = this.d.socket();
    if (socket === null) return;
    const from = f.lastSeq + 1;
    f.stream = openStream(socket, `${sessionPath(f.id)}/events?from=${from}`, {
      frames: (frames) => this.frames(f, frames),
      ended: (end) => this.ended(f, end, from),
    });
  }

  /** One chunk's events, as one batch: the renderer's fold ignores a seq it has seen. */
  private frames(f: Followed, frames: SseFrame[]): void {
    const events = frames.map((frame) => {
      const event = decodeData(frame.data);
      const seq = seqOf(frame, event);
      if (seq !== null && seq > f.lastSeq) f.lastSeq = seq;
      return event;
    });
    // Events arriving prove the stream works; a stream that only opens and drops does not.
    f.backoff.reset();
    this.d.events({ sessionId: f.id, events });
  }

  private ended(f: Followed, end: StreamEnded, from: number): void {
    f.stream = null;
    if (!this.current(f)) return;
    if (end.kind === 'refused') {
      // A refusal is the daemon's answer, and asking again gets the same one.
      this.followed.delete(f.id);
      this.d.ended({ sessionId: f.id, reason: endReason(f.id, from, end.refusal) });
      return;
    }
    void this.d.dropped().then((up) => {
      // Down, it resumes when the daemon answers again (resumeAll); unfollowed, it is gone.
      if (!up || !this.current(f) || f.stream !== null || f.retry !== null) return;
      f.retry = setTimeout(() => {
        f.retry = null;
        if (this.current(f) && f.stream === null) this.open(f);
      }, f.backoff.next());
    });
  }
}
