import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { noteText } from '../../src/renderer/text';
import { paneView } from '../../src/state/pane';
import type { Note } from '../../src/state/session';
import { rowOfConversation, type Row } from '../../src/state/sidebar';
import { log } from './helpers';

const now = Date.parse('2026-10-14T10:10:00Z');

function row(state = 'running'): Row {
  return rowOfConversation({
    id: 's-1',
    title: 's-1',
    workspace: '/w/dcode',
    state,
    live: true,
    turns: 1,
    started: '2026-10-14T10:00:00Z',
    last_activity: '2026-10-14T10:05:00Z',
    last_event: 'done.checked',
  });
}

function check(over: Partial<P.DoneChecked>): P.DoneChecked {
  return { turn_id: 't1', cycle: 1, moved: 'first', stall: 0, stall_limit: 3, ...over };
}

function notes(entries: { kind: string; note?: Note }[]): Note[] {
  return entries.flatMap((e) => (e.kind === 'note' && e.note ? [e.note] : []));
}

describe('the criteria lights move with each check (D54)', () => {
  it('show the last check of the turn under way, which cycle it is and how long it has stalled', () => {
    const l = log('s-1', '2026-10-14T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'implemente o carrinho' })
      .add('done.checked', check({ met: ['unit'], unmet: ['lint', 'e2e'] }));
    let p = paneView(row(), l.fold(), now);
    expect(p.lights).toEqual([
      { name: 'unit', state: 'met' },
      { name: 'lint', state: 'unmet' },
      { name: 'e2e', state: 'unmet' },
    ]);
    expect(p.cycle).toEqual({ n: 1, stall: 0, limit: 3 });
    expect(p.doneNote).toBeNull();

    l.add('done.checked', check({ cycle: 2, met: ['unit', 'lint'], unmet: ['e2e'], moved: 'forward' })).add(
      'done.checked',
      check({ cycle: 3, met: ['unit', 'lint'], unmet: ['e2e'], unavailable: ['a11y'], moved: 'stalled', stall: 1, touched_protected: ['cart_test.go'] }),
    );
    p = paneView(row(), l.fold(), now);
    expect(p.lights).toEqual([
      { name: 'unit', state: 'met' },
      { name: 'lint', state: 'met' },
      { name: 'e2e', state: 'unmet' },
      { name: 'a11y', state: 'unavailable' },
    ]);
    expect(p.cycle).toEqual({ n: 3, stall: 1, limit: 3 });
    expect(p.touched).toEqual(['cart_test.go']);
  });

  it('give way to what turn.completed measured once the turn ends', () => {
    const v = log('s-1', '2026-10-14T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'implemente o carrinho' })
      .add('done.checked', check({ unmet: ['e2e'] }))
      .add('done.checked', check({ cycle: 2, met: ['e2e'], moved: 'forward' }))
      .add('turn.completed', { turn_id: 't1', reason: 'done', completion: { verification: 'passed', met: ['e2e'] } })
      .fold();
    expect(v.check).toBeNull();
    const p = paneView(row('idle'), v, now);
    expect(p.lights).toEqual([{ name: 'e2e', state: 'met' }]);
    expect(p.cycle).toBeNull();
  });

  it('start over with the next turn, and never take a check of another turn', () => {
    const v = log('s-1', '2026-10-14T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'um' })
      .add('done.checked', check({ unmet: ['e2e'] }))
      .add('turn.completed', { turn_id: 't1', reason: 'incomplete', completion: { verification: 'failed', unmet: ['e2e'] } })
      .add('turn.started', { turn_id: 't2', text: 'dois' })
      .add('done.checked', check({ turn_id: 't1', cycle: 9, met: ['e2e'] }))
      .fold();
    expect(v.check).toBeNull();
    const p = paneView(row(), v, now);
    // The last measured turn still stands until this one checks anything.
    expect(p.lights).toEqual([{ name: 'e2e', state: 'unmet' }]);
    expect(p.cycle).toBeNull();
  });

  it('say in the flow what a check that broke a criterion put back', () => {
    const v = log('s-1', '2026-10-14T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'implemente o carrinho' })
      .add('done.checked', check({ met: ['unit', 'e2e'], unmet: ['a11y'] }))
      .add(
        'done.checked',
        check({ cycle: 2, met: ['unit'], unmet: ['a11y', 'e2e'], moved: 'backward', stall: 1, undone: ['src/cart.ts'] }),
      )
      .fold();
    const back = notes(v.entries).filter((n) => n.kind === 'cycle-back');
    expect(back).toEqual([{ kind: 'cycle-back', cycle: 2, broke: ['e2e'], undone: ['src/cart.ts'] }]);
    expect(noteText(back[0] as Note)).toEqual({ text: '⟲ Ciclo 2 voltou: e2e quebrou, desfeito src/cart.ts', tone: 'warn' });
    expect(noteText({ kind: 'cycle-back', cycle: 4, broke: ['a', 'b'], undone: [] }).text).toBe(
      '⟲ Ciclo 4 voltou: a, b quebraram, nada pôde ser desfeito',
    );
  });

  it('add nothing to the flow for a check that moved forward or stalled', () => {
    const v = log('s-1', '2026-10-14T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'x' })
      .add('done.checked', check({ unmet: ['e2e'] }))
      .add('done.checked', check({ cycle: 2, unmet: ['e2e'], moved: 'stalled', stall: 1 }))
      .fold();
    expect(notes(v.entries)).toEqual([]);
  });
});
