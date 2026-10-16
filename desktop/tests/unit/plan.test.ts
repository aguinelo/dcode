import { describe, expect, it } from 'vitest';
import { pinnedPlan, withoutPinned } from '../../src/state/plan';
import { created, log } from './helpers';

const items = (...statuses: string[]) => statuses.map((status, i) => ({ id: i + 1, text: `passo ${i + 1}`, status }));

describe('the plan pinned at the top of the turn (D57)', () => {
  it('pins the plan the running turn wrote, with the step under way', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'Lê o PDF e escreve a análise.' })
      .add('plan.updated', { items: items('done', 'active', 'pending') })
      .fold();
    const pin = pinnedPlan(v);
    expect(pin).toMatchObject({ step: 2, total: 3, current: 'passo 2' });
    expect(pin?.items.map((i) => i.status)).toEqual(['done', 'active', 'pending']);
  });

  it('follows each revision, and takes the first unfinished step when none is active', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('active', 'pending', 'pending') })
      .add('plan.updated', { items: items('done', 'done', 'pending') })
      .fold();
    expect(pinnedPlan(v)).toMatchObject({ step: 3, total: 3, current: 'passo 3' });
  });

  it('stands at the last step once every step is done', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('done', 'done') })
      .fold();
    expect(pinnedPlan(v)).toMatchObject({ step: 2, total: 2, current: 'passo 2' });
  });

  it('stays pinned while the turn waits on an approval', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('active') })
      .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c1', name: 'bash', input: { command: 'npm i' } })
      .add('tool.approval_required', {
        approval_id: 'a-1',
        turn_id: 't-1',
        tool_call_id: 'c1',
        tool: 'bash',
        command: 'npm i',
        boundary_crossed: 'network',
        expires_at: '0001-01-01T00:00:00Z',
        reason: 'network',
      })
      .fold();
    expect(v.state).toBe('blocked');
    expect(pinnedPlan(v)).not.toBeNull();
  });

  it('goes back into the flow when the turn ends', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('done', 'done') })
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .fold();
    expect(pinnedPlan(v)).toBeNull();
    expect(withoutPinned(v.entries, pinnedPlan(v)).some((e) => e.kind === 'plan')).toBe(true);
  });

  it('leaves a plan of an earlier turn in the flow, not pinned over the new one', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('done') })
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .add('turn.started', { turn_id: 't-2', text: 'y' })
      .fold();
    expect(pinnedPlan(v)).toBeNull();
  });

  it('pins nothing for an empty plan', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: [] })
      .fold();
    expect(pinnedPlan(v)).toBeNull();
  });

  it('takes the pinned plan out of the flow, and only it', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1', text: 'x' })
      .add('plan.updated', { items: items('active') })
      .fold();
    const rest = withoutPinned(v.entries, pinnedPlan(v));
    expect(rest.some((e) => e.kind === 'plan')).toBe(false);
    expect(rest.some((e) => e.kind === 'user')).toBe(true);
  });
});
