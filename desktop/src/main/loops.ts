import { LoopAwaitingReview, LoopChanged, LoopQualifying, LoopSnapshot, LoopSurveying, LoopWorking, Version, type LoopRun } from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { LoopsPush } from '../shared/api';
import { Backoff } from './follow';
import { decodeData } from './sse';
import { openStream, shown, type Stream, type StreamEnded } from './wire';

// The `/loop` runs the daemon conducts (D55), as the main process keeps them:
// folded from their stream, like the list of conversations (D21), so a window
// that opens — or reloads — after the stream did starts from every run.
//
// Only what folding needs is checked: a kind, and an id on each run. The
// renderer decodes every frame against the generated types and says what does
// not match; a frame this fold cannot read is passed on all the same.

export type FoldedLoops = { ok: true; list: LoopRun[] } | { ok: false; problem: string };

function hasId(v: unknown): v is LoopRun {
  return isRecord(v) && typeof v.id === 'string' && v.id !== '';
}

/**
 * Applies one frame: a snapshot replaces the list, `changed` puts the run in
 * its place — or last, when it is new, since the daemon lists the oldest
 * first. Before the first snapshot there is no list to change.
 */
export function foldLoops(list: LoopRun[] | null, frame: unknown): FoldedLoops {
  if (!isRecord(frame) || typeof frame.kind !== 'string') return { ok: false, problem: 'not a change with a kind' };
  switch (frame.kind) {
    case LoopSnapshot: {
      const all = frame.loops ?? [];
      if (!Array.isArray(all) || !all.every(hasId)) return { ok: false, problem: 'a snapshot without a list of runs with ids' };
      return { ok: true, list: [...all] };
    }
    case LoopChanged: {
      if (!hasId(frame.loop)) return { ok: false, problem: 'a change without a run with an id' };
      if (list === null) return { ok: false, problem: 'a change before the first snapshot' };
      const changed = frame.loop;
      const at = list.findIndex((r) => r.id === changed.id);
      return { ok: true, list: at < 0 ? [...list, changed] : list.map((r, i) => (i === at ? changed : r)) };
    }
    default:
      return { ok: false, problem: `an unknown kind “${frame.kind}”` };
  }
}

const LIVE: readonly string[] = [LoopSurveying, LoopQualifying, LoopAwaitingReview, LoopWorking];

/** Runs that have not ended: stopping the daemon would end them. */
export function countLiveLoops(list: readonly LoopRun[]): number {
  return list.filter((r) => LIVE.includes(r.state)).length;
}

const LOOP_EVENTS = `/${Version}/loops/events`;

export interface LoopListDeps {
  retryFromMs: number;
  retryUpToMs: number;
  push(p: LoopsPush): void;
  /** The stream dropped: asks /health, and resolves true when the daemon still answers. */
  dropped(): Promise<boolean>;
  log(line: string): void;
}

/**
 * The stream of loops, opened when the daemon answers and closed when it is
 * lost. A daemon that refuses it — one older than `/loop` in the daemon — is
 * said once, and the window goes on without loops: the conversations do not
 * depend on them.
 */
export class LoopList {
  private list: LoopRun[] | null = null;
  private refused: string | null = null;
  private stream: Stream | null = null;
  private retry: ReturnType<typeof setTimeout> | null = null;
  /** Open was asked and close was not since: a drop is retried only then. */
  private wanted = false;
  private readonly backoff: Backoff;

  constructor(private readonly d: LoopListDeps) {
    this.backoff = new Backoff(d.retryFromMs, d.retryUpToMs);
  }

  /** What a window that opens now starts from: every run, or why there are none to give; null before either. */
  now(): LoopsPush | null {
    if (this.refused !== null) return { kind: 'refused', reason: this.refused };
    return this.list === null ? null : { kind: 'frame', frame: { kind: LoopSnapshot, loops: this.list } };
  }

  /** Runs still alive, as last listed. */
  live(): number {
    return this.list === null ? 0 : countLiveLoops(this.list);
  }

  open(socket: string): void {
    this.close();
    this.wanted = true;
    this.refused = null;
    this.stream = openStream(socket, LOOP_EVENTS, {
      frames: (frames) => {
        this.backoff.reset();
        for (const f of frames) this.frame(decodeData(f.data));
      },
      ended: (end) => this.ended(socket, end),
    });
  }

  close(): void {
    this.wanted = false;
    this.stream?.close();
    this.stream = null;
    if (this.retry) clearTimeout(this.retry);
    this.retry = null;
  }

  private frame(frame: unknown): void {
    const folded = foldLoops(this.list, frame);
    if (folded.ok) this.list = folded.list;
    else this.d.log(`a frame of the list of loops was not folded (${folded.problem}): ${shown(frame)}`);
    this.d.push({ kind: 'frame', frame });
  }

  private ended(socket: string, end: StreamEnded): void {
    this.stream = null;
    if (!this.wanted) return;
    if (end.kind === 'refused') {
      // The daemon's answer, and asking again gets the same one.
      const older = end.status === 404 || end.refusal.code === 'not_found';
      this.refused = older
        ? `Este dcode não conduz loops (GET /v1/loops/events respondeu ${end.status}): o /loop pela janela precisa de um dcode mais novo (dcode update).`
        : `O daemon recusou a lista de loops: ${end.refusal.message}`;
      this.d.push({ kind: 'refused', reason: this.refused });
      return;
    }
    this.d.log(`the list of loops closed (${end.cause}); asking /health`);
    void this.d.dropped().then((up) => {
      if (!up || !this.wanted || this.stream !== null || this.retry !== null) return;
      this.retry = setTimeout(() => {
        this.retry = null;
        if (this.wanted && this.stream === null) this.open(socket);
      }, this.backoff.next());
    });
  }
}
