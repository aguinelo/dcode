import { afterEach, describe, expect, it } from 'vitest';
import { Connection } from '../../src/main/connection';
import type { LoopsPush } from '../../src/shared/api';
import { fakeDaemon, tempSocket, until, type FakeOptions } from './fake-daemon';

// The stream of loops (D55), next to the list of conversations: passed on as
// it came, kept for a window that opens later, and — on a daemon older than
// /loop in the daemon — said once, without costing the window its daemon.

const cleanups: (() => void | Promise<void>)[] = [];
afterEach(async () => {
  for (const c of cleanups.splice(0).reverse()) await c();
});

async function connect(o: FakeOptions) {
  const t = tempSocket();
  cleanups.push(t.remove);
  const d = await fakeDaemon(t.socket, o);
  cleanups.push(() => d.kill());
  const loops: LoopsPush[] = [];
  let state = '';
  const conn = new Connection(
    {
      lookup: {
        env: { DCODE_SOCKET: t.socket },
        home: '/nowhere',
        cwd: '/',
        probe: () => {
          throw new Error('looked for a binary to attach');
        },
        run: () => Promise.reject(new Error('ran a command to attach')),
      },
      serve: () => {
        throw new Error('started a daemon while one answers');
      },
      timing: { healthEveryMs: 60_000, retryFromMs: 20, retryUpToMs: 40 },
    },
    { status: (s) => (state = s.state), conversations: () => {}, sessionEvents: () => {}, streamEnd: () => {}, loops: (p) => loops.push(p) },
  );
  cleanups.push(() => conn.close());
  conn.start();
  return { conn, loops, state: () => state };
}

const run = { id: 'L', workspace: '/w', state: 'working', specs: [], current: -1, started: '2026-10-14T10:00:00Z' };

describe('the stream of loops', () => {
  it('passes the snapshot on and keeps it, counting the runs alive', async () => {
    const { conn, loops } = await connect({ loops: [run] });
    await until(() => loops.length === 1, 'the snapshot of loops');
    expect(loops[0]).toEqual({ kind: 'frame', frame: { kind: 'snapshot', loops: [run] } });
    expect(conn.loops.now()).toEqual({ kind: 'frame', frame: { kind: 'snapshot', loops: [run] } });
    expect(conn.loops.live()).toBe(1);
  });

  it('says a daemon without it, once, and stays connected', async () => {
    const { conn, loops, state } = await connect({});
    await until(() => loops.length === 1, 'the refusal');
    expect(loops[0]?.kind).toBe('refused');
    expect(loops[0]?.kind === 'refused' && loops[0].reason).toContain('não conduz loops');
    expect(conn.loops.now()).toEqual(loops[0]);
    expect(state()).toBe('connected');
  });
});
