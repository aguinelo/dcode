import { describe, expect, it } from 'vitest';
import type { ImageAttachment } from '../../src/shared/api';
import { chipsOf, turnTouched } from '../../src/state/turn';
import { created, done, log } from './helpers';

const png = (name: string): ImageAttachment => ({ name, mediaType: 'image/png', data: 'AAAA', bytes: 3 });

describe('what the running turn touched, on its activity line (D58)', () => {
  it('lists the files this turn wrote, newest first, each with the call that last wrote it', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c1', name: 'write', input: { path: 'a.go' } })
      .add('tool.completed', done('c1', { added: 4 }))
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c2', name: 'read', input: { path: 'b.go' } })
      .add('tool.completed', done('c2'))
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c3', name: 'edit', input: { path: 'c.go' } })
      .add('tool.completed', done('c3', { added: 1, removed: 1 }))
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c4', name: 'edit', input: { path: 'a.go' } })
      .add('tool.completed', done('c4', { added: 2, removed: 1 }))
      .fold();
    expect(turnTouched(v, new Map()).files).toEqual([
      { path: 'a.go', added: 6, removed: 1, failed: false, callId: 'c4' },
      { path: 'c.go', added: 1, removed: 1, failed: false, callId: 'c3' },
    ]);
  });

  it('leaves out what an earlier turn wrote', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c1', name: 'write', input: { path: 'old.go' } })
      .add('tool.completed', done('c1', { added: 4 }))
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .add('turn.started', { turn_id: 't-2', text: 'y' })
      .add('tool.requested', { turn_id: 't-2', tool_call_id: 'c2', name: 'write', input: { path: 'new.go' } })
      .fold();
    expect(turnTouched(v, new Map()).files.map((f) => f.path)).toEqual(['new.go']);
  });

  it('touches nothing outside a turn', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c1', name: 'write', input: { path: 'a.go' } })
      .add('tool.completed', done('c1'))
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .fold();
    expect(turnTouched(v, new Map())).toEqual({ files: [], images: [] });
  });

  it('carries the pictures sent with this turn’s message, not an earlier one’s', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'antes' })
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .add('turn.started', { turn_id: 't-2', text: 'olha isto' })
      .fold();
    const first = v.entries.find((e) => e.kind === 'user' && e.text === 'antes')!.seq;
    const second = v.entries.find((e) => e.kind === 'user' && e.text === 'olha isto')!.seq;
    const imagesAt = new Map([
      [first, [png('velha.png')]],
      [second, [png('tela.png'), png('erro.png')]],
    ]);
    expect(turnTouched(v, imagesAt).images.map((i) => i.name)).toEqual(['tela.png', 'erro.png']);
  });
});

describe('the chips that fit on the line', () => {
  it('shows up to the room and counts the rest', () => {
    expect(chipsOf(['a', 'b', 'c', 'd', 'e'], 3)).toEqual({ shown: ['a', 'b', 'c'], more: 2 });
  });

  it('shows everything when it fits, with nothing more', () => {
    expect(chipsOf(['a', 'b'], 3)).toEqual({ shown: ['a', 'b'], more: 0 });
  });
});
