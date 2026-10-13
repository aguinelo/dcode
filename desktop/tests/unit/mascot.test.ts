import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { MASCOT_WORDS, mascotOf } from '../../src/state/mascot';
import { paneView } from '../../src/state/pane';
import { rowOfConversation, type Row } from '../../src/state/sidebar';
import { created, done, log } from './helpers';

function row(id: string, over: Partial<P.Conversation> = {}): Row {
  return rowOfConversation({
    id,
    title: id,
    workspace: '/w/dcode',
    state: 'idle',
    live: true,
    turns: 1,
    started: '2026-10-01T10:00:00Z',
    last_activity: '2026-10-01T10:05:00Z',
    last_event: 'turn.completed',
    ...over,
  });
}

const now = Date.parse('2026-09-28T18:00:00.000Z');

describe("the mascot's state", () => {
  it('waits for you before anything else', () => {
    expect(mascotOf('blocked', 'failed', 'writing')).toBe('esperando');
  });

  it('plans while the model thinks or reads, codes while it writes or delegates, and verifies while a command runs', () => {
    expect(mascotOf('running', undefined, null)).toBe('planejando');
    expect(mascotOf('running', undefined, 'reading')).toBe('planejando');
    expect(mascotOf('running', undefined, 'writing')).toBe('codando');
    expect(mascotOf('running', undefined, 'delegating')).toBe('codando');
    expect(mascotOf('running', undefined, 'other')).toBe('codando');
    expect(mascotOf('running', undefined, 'running')).toBe('verificando');
  });

  it('says what the last measured turn established, and nothing it did not', () => {
    expect(mascotOf('idle', 'passed', null)).toBe('deu-certo');
    expect(mascotOf('idle', 'failed', null)).toBe('bug');
    expect(mascotOf('idle', 'stale', null)).toBe('hum');
    expect(mascotOf('idle', 'unavailable', null)).toBe('hum');
    expect(mascotOf('idle', undefined, null)).toBe('parado');
    expect(mascotOf('recorded', '', null)).toBe('parado');
  });

  it('has a word for each state, the sheet\'s where it has one', () => {
    expect(MASCOT_WORDS).toEqual({
      planejando: 'Planejando…',
      codando: 'Codando…',
      verificando: 'Verificando…',
      'deu-certo': 'Deu certo!',
      bug: 'Bug encontrado…',
      esperando: 'Esperando você',
      hum: 'Hum…',
      parado: 'Parado',
    });
  });

  it('is what a window shows of its conversation, from the call running now', () => {
    const writing = log('s1', '2026-09-28T17:00:00.000Z')
      .add('session.created', created)
      .add('turn.started', { turn_id: 't', text: 'conserte o teste' })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'edit', input: { path: 'a.go' } });
    expect(paneView(row('s1', { state: 'running' }), writing.fold(), now).mascot).toBe('codando');
    const tests = writing
      .add('tool.completed', done('c1'))
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c2', name: 'bash', input: { command: 'go test ./...' } });
    expect(paneView(row('s1', { state: 'running' }), tests.fold(), now).mascot).toBe('verificando');
    expect(paneView(row('s2', { state: 'blocked' }), undefined, now).mascot).toBe('esperando');
    expect(paneView(row('s3', { verification: 'passed' }), undefined, now).mascot).toBe('deu-certo');
  });
});
