import { describe, expect, it } from 'vitest';
import { noteText } from '../../src/renderer/text';
import { applyEvent, emptySession } from '../../src/state/session';
import { created, done, log } from './helpers';

describe('a turn', () => {
  it('starts running with the question in the flow, and ends idle with the context measured', () => {
    const l = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'Cataloga as chamadas.' });
    const running = l.fold();
    expect(running.state).toBe('running');
    expect(running.turn).toEqual({ id: 't-1', startedAt: l.events[1]?.at });
    expect(running.entries).toMatchObject([{ kind: 'user', text: 'Cataloga as chamadas.', steer: false }]);

    const idle = l
      .add('turn.completed', {
        turn_id: 't-1',
        reason: 'done',
        usage: { input_tokens: 90000, output_tokens: 1200, context_tokens: 82000 },
      })
      .fold();
    expect(idle.state).toBe('idle');
    expect(idle.turn).toBeNull();
    expect(idle.turns).toBe(1);
    // The daemon's measure of the context, never the cumulative input.
    expect(idle.contextTokens).toBe(82000);
  });

  it('keeps streamed text in one message, and a correction as the person speaking', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('message.reasoning', { turn_id: 't-1', text: 'pensando ' })
      .add('message.reasoning', { turn_id: 't-1', text: 'mais' })
      .add('message.delta', { turn_id: 't-1', text: 'São quatro ' })
      .add('message.delta', { turn_id: 't-1', text: 'pacotes.' })
      .add('turn.steered', { turn_id: 't-1', text: 'só o alpha' })
      .fold();
    expect(v.entries.map((e) => e.kind)).toEqual(['reasoning', 'model', 'user']);
    expect(v.entries[0]).toMatchObject({ text: 'pensando mais' });
    expect(v.entries[1]).toMatchObject({ text: 'São quatro pacotes.' });
    expect(v.entries[2]).toMatchObject({ text: 'só o alpha', steer: true });
  });

  it('opens a new message each time the turn goes round again', () => {
    const said = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('message.delta', { turn_id: 't-1', text: 'Corrigi o parser.' })
      .add('progress', { turn_id: 't-1', kind: 'rounds', done: 1, total: 100 })
      .add('message.delta', { turn_id: 't-1', text: 'O teste ainda falha.' })
      .fold();
    expect(said.entries.flatMap((e) => (e.kind === 'model' ? [e.text] : []))).toEqual([
      'Corrigi o parser.',
      'O teste ainda falha.',
    ]);

    const thought = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('message.reasoning', { turn_id: 't-1', text: 'falta o parser' })
      .add('progress', { turn_id: 't-1', kind: 'rounds', done: 1, total: 100 })
      .add('message.reasoning', { turn_id: 't-1', text: 'o teste falhou' })
      .fold();
    expect(thought.entries.flatMap((e) => (e.kind === 'reasoning' ? [e.text] : []))).toEqual([
      'falta o parser',
      'o teste falhou',
    ]);
  });
});

describe('tool calls', () => {
  it('runs, then carries its result, and sums what the tools reported they changed', () => {
    const v = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'a.json' } })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c2', name: 'edit', input: { path: 'b.go' } })
      .add('tool.completed', done('c2', { added: 3, removed: 1 }))
      .add('tool.completed', done('c1', { added: 54 }))
      .fold();
    const calls = v.entries.flatMap((e) => (e.kind === 'tool' ? [e.call] : []));
    expect(calls.map((c) => [c.id, c.status])).toEqual([
      ['c1', 'ok'],
      ['c2', 'ok'],
    ]);
    expect([v.added, v.removed]).toEqual([57, 1]);
  });

  it('draws a call from the moment it starts arriving, and fills the same line once requested', () => {
    const v = log()
      .add('progress', { turn_id: 't', tool_call_id: 'c1', name: 'write', kind: 'arguments', done: 512 })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'x' } })
      .fold();
    expect(v.entries).toHaveLength(1);
    expect(v.entries[0]).toMatchObject({ kind: 'tool', call: { status: 'running', input: { path: 'x' } } });
  });

  it('says so when a result answers a call nobody announced', () => {
    const v = log().add('tool.completed', done('ghost')).fold();
    expect(v.entries).toMatchObject([{ kind: 'note', note: { kind: 'orphan-result', callId: 'ghost' } }]);
  });
});

describe('an approval', () => {
  const asked = () =>
    log()
      .add('turn.started', { turn_id: 't-1' })
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c3', name: 'bash', input: { command: 'npm i' } })
      .add('tool.approval_required', {
        approval_id: 't-1-2',
        turn_id: 't-1',
        tool_call_id: 'c3',
        tool: 'bash',
        command: 'npm i',
        boundary_crossed: 'network',
        expires_at: '0001-01-01T00:00:00Z',
      });

  it('blocks the session and stays pending until answered', () => {
    const v = asked().fold();
    expect(v.state).toBe('blocked');
    expect(v.pendingApprovalId).toBe('t-1-2');
    expect(v.entries.at(-1)).toMatchObject({ kind: 'approval', decision: null });
  });

  it('keeps the question in the flow with its answer, matched by id', () => {
    const v = asked().add('tool.approval_resolved', { approval_id: 't-1-2', decision: 'deny' }).fold();
    expect(v.state).toBe('running');
    expect(v.pendingApprovalId).toBeNull();
    expect(v.entries.find((e) => e.kind === 'approval')).toMatchObject({ decision: 'deny' });
  });
});

describe('the log', () => {
  it('ignores an event it already has, and says so when events are missing', () => {
    const l = log().add('turn.started', { turn_id: 't' }).add('message.delta', { turn_id: 't', text: 'a' });
    const once = l.fold();
    const replayed = l.events.reduce(applyEvent, once);
    expect(replayed).toBe(once);

    const gap = applyEvent(once, {
      seq: 5,
      session_id: 's-1',
      type: 'message.delta',
      at: '2026-09-28T17:00:09.000Z',
      payload: { turn_id: 't', text: 'b' },
    });
    expect(gap.entries.find((e) => e.kind === 'note')).toMatchObject({ note: { kind: 'gap', from: 3, to: 4 } });
    expect(gap.lastSeq).toBe(5);
  });

  it('ends a turn with its seal, or with why it stopped short', () => {
    const sealed = log()
      .add('turn.started', { turn_id: 't' })
      .add('turn.completed', { turn_id: 't', reason: 'done', completion: { verification: 'passed', met: ['go test'] } })
      .fold();
    expect(sealed.entries.at(-1)).toMatchObject({ note: { kind: 'completion' } });

    const clean = log()
      .add('turn.completed', { turn_id: 't', reason: 'done', completion: { verification: 'clean' } })
      .fold();
    expect(clean.entries).toEqual([]);

    const ceiling = log()
      .add('turn.started', { turn_id: 't' })
      .add('progress', { turn_id: 't', kind: 'rounds', done: 100, total: 100 })
      .add('turn.completed', { turn_id: 't', reason: 'max_iterations' })
      .fold();
    expect(ceiling.entries.at(-1)).toMatchObject({ note: { kind: 'stopped', reason: 'max_iterations', rounds: 100 } });
  });

  it('takes the session facts from session.created and follows a mode switch', () => {
    const v = log()
      .add('session.created', created)
      .add('session.mode_changed', { mode: 'assist', sandbox_mode: 'workspace-write' })
      .add('session.mode_changed', { previous: 'assist', mode: 'plan', sandbox_mode: 'read-only' })
      .fold(emptySession('s-1'));
    expect(v.info?.branch).toBe('feat/x');
    expect([v.mode, v.sandbox]).toEqual(['plan', 'read-only']);
    // The first announce is where the session starts, not a switch.
    expect(v.entries.filter((e) => e.kind === 'note')).toHaveLength(1);
  });

  it('says what the session said as it opens, as a warning in the flow and not an error', () => {
    const said = 'using the claude family: nobody measured it';
    const v = log()
      .add('session.created', created)
      .add('session.notice', { code: 'family_unmeasured', message: said })
      .fold(emptySession('s-1'));
    const last = v.entries.at(-1);
    expect(last).toMatchObject({ kind: 'note', note: { kind: 'notice', code: 'family_unmeasured', message: said } });
    if (last?.kind !== 'note') throw new Error('the notice is not a note');
    expect(noteText(last.note)).toEqual({ text: said, tone: 'warn' });
  });
});
