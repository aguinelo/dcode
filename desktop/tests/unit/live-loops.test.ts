import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { LiveStore } from '../../src/renderer/live';
import type { Answer, DcodeApi, LoopsPush } from '../../src/shared/api';

// The loops the window draws, and what it asks the daemon about them (D55),
// through the main process.

function loopsApi() {
  const asked: string[] = [];
  const pushes: ((p: LoopsPush) => void)[] = [];
  const answers: Record<string, Answer<unknown>> = {};
  const answer = (key: string, what: string): Promise<Answer<unknown>> => {
    asked.push(what);
    return Promise.resolve(answers[key] ?? { ok: false, refusal: { code: 'internal', message: 'not under test' } });
  };
  const quiet = () => () => {};
  const api = {
    onDaemon: quiet,
    onConversations: quiet,
    onSessionEvents: quiet,
    onStreamEnd: quiet,
    loops: {
      on: (l: (p: LoopsPush) => void) => (pushes.push(l), () => {}),
      start: (ws: string, args: string) => answer('start', `start in ${ws}: ${args}`),
      stop: (id: string, when: string) => answer('stop', `stop ${id} ${when}`),
      specs: (ws: string) => answer('specs', `specs of ${ws}`),
    },
  } as unknown as DcodeApi;
  return { api, asked, answers, push: (p: LoopsPush) => pushes.forEach((l) => l(p)) };
}

const run: P.LoopRun = { id: 'L', workspace: '/w', state: 'surveying', specs: [], current: -1, started: '2026-10-14T10:00:00Z' };

function started() {
  const f = loopsApi();
  const store = new LiveStore(f.api);
  const notices: string[] = [];
  store.onNotice((t) => notices.push(t));
  store.start();
  return { f, store, notices };
}

describe('the loops the window draws', () => {
  it('folds the list the main process pushes', () => {
    const { f, store } = started();
    expect(store.getSnapshot().loops.loaded).toBe(false);
    f.push({ kind: 'frame', frame: { kind: 'snapshot', loops: [run] } });
    f.push({ kind: 'frame', frame: { kind: 'changed', loop: { ...run, state: 'working' } } });
    expect(store.getSnapshot().loops.byId.L?.state).toBe('working');
  });

  it('says a frame it cannot read, and a daemon that has no loops to give', () => {
    const { f, store, notices } = started();
    f.push({ kind: 'frame', frame: { kind: 'changed', loop: { id: 'L' } } });
    f.push({ kind: 'refused', reason: 'Este dcode não serve a lista de loops.' });
    expect(notices[0]).toMatch(/^Mudança ilegível na lista de loops: /);
    expect(notices[1]).toBe('Este dcode não serve a lista de loops.');
    expect(store.getSnapshot().loops.problem).toBe('Este dcode não serve a lista de loops.');
  });
});

describe('what the window asks about a loop', () => {
  it('starts one with the argument as typed, and answers the run', async () => {
    const { f, store } = started();
    f.answers.start = { ok: true, value: run };
    expect(await store.actions.startLoop('/w', 'specs/a --protect t/**')).toEqual({ ok: true, value: run });
    expect(f.asked).toEqual(['start in /w: specs/a --protect t/**']);
  });

  it('says why one did not start, with the daemon’s reason', async () => {
    const { f, store } = started();
    f.answers.start = { ok: false, refusal: { code: 'loop_already_running', message: 'loop K is already running in /w' } };
    const got = await store.actions.startLoop('/w', 'specs/a');
    expect(got).toEqual({ ok: false, why: 'Já há um loop vivo neste projeto, e outro não começa enquanto ele roda: loop K is already running in /w' });
  });

  it('stops one now or after its folder, and says a refusal', async () => {
    const { f, store } = started();
    f.answers.stop = { ok: false, refusal: { code: 'loop_not_found', message: 'no loop L' } };
    expect(await store.actions.stopLoop('L', 'after_spec')).toEqual({ ok: false, why: 'O loop não parou: no loop L' });
    expect(f.asked).toEqual(['stop L after_spec']);
  });

  it('reads the folders a project declares, and says an answer it cannot read', async () => {
    const { f, store } = started();
    const folder = { path: 'specs/a', criteria: 2, unmet: 0, pending: false, measured: false };
    f.answers.specs = { ok: true, value: { specs: [folder] } };
    expect(await store.actions.listSpecs('/w')).toEqual({ ok: true, value: [folder] });
    f.answers.specs = { ok: true, value: { specs: 'x' } };
    expect(await store.actions.listSpecs('/w')).toMatchObject({ ok: false, why: expect.stringMatching(/^Resposta ilegível do daemon: /) });
  });
});
