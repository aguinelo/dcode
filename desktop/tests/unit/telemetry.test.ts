import { describe, expect, it } from 'vitest';
import { tokens, turnTelemetry, usageFact } from '../../src/state/telemetry';
import { created, log } from './helpers';

const rounds = (done: number, total = 100) => ({ turn_id: 't-1', kind: 'rounds', done, total });
const inFlight = (done: number, total = 4) => ({ turn_id: 't-1', kind: 'in_flight', done, total });

describe('the usage a turn reports', () => {
  it('is kept whole from turn.completed, input and cache included', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't-1' })
      .add('turn.completed', {
        turn_id: 't-1',
        reason: 'done',
        usage: { input_tokens: 90000, output_tokens: 1200, cache_read_tokens: 80000, cache_write_tokens: 3000, context_tokens: 82000 },
      })
      .fold();
    expect(v.usage).toEqual({
      input_tokens: 90000,
      output_tokens: 1200,
      cache_read_tokens: 80000,
      cache_write_tokens: 3000,
      context_tokens: 82000,
    });
  });

  it('belongs to the turn that reported it: a new turn starts without one', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('turn.completed', { turn_id: 't-1', reason: 'done', usage: { input_tokens: 10, output_tokens: 5 } })
      .add('turn.started', { turn_id: 't-2' })
      .fold();
    expect(v.usage).toBeNull();
  });

  it('is absent when the provider did not report it, never the last turn carried over', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('turn.completed', { turn_id: 't-1', reason: 'done', usage: { input_tokens: 10, output_tokens: 5 } })
      .add('turn.started', { turn_id: 't-2' })
      .add('turn.completed', { turn_id: 't-2', reason: 'done' })
      .fold();
    expect(v.usage).toBeNull();
    expect(v.turns).toBe(2);
  });
});

describe('what the window says of a running turn', () => {
  it('is nothing at all when no turn runs', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('progress', rounds(60))
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .fold();
    expect(turnTelemetry(v)).toBeNull();
  });

  it('counts, but draws nothing, far from every ceiling', () => {
    const v = log().add('turn.started', { turn_id: 't-1' }).add('progress', rounds(2)).add('progress', inFlight(2)).fold();
    const t = turnTelemetry(v);
    expect(t?.rounds).toBeNull();
    expect(t?.inFlight).toBeNull();
    // The full count is still there to read on hover.
    expect(t?.detail).toBe('rodada 2 de 100 · 2 de 4 chamadas em voo');
  });

  it('draws the rounds from half the ceiling, warning, and as an error from three quarters', () => {
    const at = (n: number) => turnTelemetry(log().add('turn.started', { turn_id: 't-1' }).add('progress', rounds(n)).fold());
    expect(at(49)?.rounds).toBeNull();
    expect(at(50)?.rounds).toEqual({ text: 'rodada 50/100', tone: 'warn' });
    expect(at(74)?.rounds).toEqual({ text: 'rodada 74/100', tone: 'warn' });
    expect(at(75)?.rounds).toEqual({ text: 'rodada 75/100', tone: 'err' });
  });

  it('draws the calls in flight only when every slot is taken', () => {
    const at = (n: number) => turnTelemetry(log().add('turn.started', { turn_id: 't-1' }).add('progress', inFlight(n)).fold());
    expect(at(3)?.inFlight).toBeNull();
    expect(at(4)?.inFlight).toEqual({ text: '4/4 em voo', tone: 'warn' });
  });

  it('says a turn with no ceiling by its count alone, and draws nothing for it', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('progress', { turn_id: 't-1', kind: 'rounds', done: 7 })
      .fold();
    const t = turnTelemetry(v);
    expect(t?.rounds).toBeNull();
    expect(t?.detail).toBe('rodada 7');
  });

  it('says plainly when the daemon has not counted anything yet', () => {
    const t = turnTelemetry(log().add('turn.started', { turn_id: 't-1' }).fold());
    expect(t?.detail).toBe('a primeira rodada ainda não foi contada');
  });

  it('starts every turn from its own counts', () => {
    const v = log()
      .add('turn.started', { turn_id: 't-1' })
      .add('progress', rounds(90))
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .add('turn.started', { turn_id: 't-2' })
      .fold();
    expect(turnTelemetry(v)?.rounds).toBeNull();
  });
});

describe('tokens as they read on screen', () => {
  it('reads like the TUI, with the comma the interface writes', () => {
    expect(tokens(0)).toBe('0');
    expect(tokens(820)).toBe('820');
    expect(tokens(2400)).toBe('2,4k');
    expect(tokens(183000)).toBe('183,0k');
    expect(tokens(1_500_000)).toBe('1,5M');
  });

  it('puts the output first and the input and cache in the detail', () => {
    expect(usageFact({ input_tokens: 90000, output_tokens: 1200, cache_read_tokens: 80000, cache_write_tokens: 3000 })).toEqual({
      text: '1,2k de saída',
      detail: '1,2k de saída · 90,0k de entrada, somada pelas rodadas · 80,0k lidos do cache · 3,0k escritos no cache',
    });
    expect(usageFact({ input_tokens: 10, output_tokens: 5 })).toEqual({
      text: '5 de saída',
      detail: '5 de saída · 10 de entrada, somada pelas rodadas',
    });
  });
});
