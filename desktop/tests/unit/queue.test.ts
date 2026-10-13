import { describe, expect, it } from 'vitest';
import { dropAt, dropLast, enqueue, failed, noQueues, resume, sent, step, type Queues } from '../../src/state/queue';

// The messages queued while a turn runs (D47): kept in order, sent one at a
// time when the conversation is idle again, each only after the turn the one
// before started has ended. A send that fails stops the queue and says why.

const idle = (turns: number) => ({ state: 'idle', turns });
const running = (turns: number) => ({ state: 'running', turns });

describe('queuing', () => {
  it('keeps the messages in order, per conversation', () => {
    let q: Queues = noQueues;
    q = enqueue(q, 's-1', 'primeiro');
    q = enqueue(q, 's-1', 'segundo');
    q = enqueue(q, 's-2', 'outro');
    expect(q['s-1']?.items).toEqual(['primeiro', 'segundo']);
    expect(q['s-2']?.items).toEqual(['outro']);
  });

  it('^X drops the last queued; × drops the one it is on; an empty queue goes', () => {
    let q = enqueue(enqueue(enqueue(noQueues, 's-1', 'a'), 's-1', 'b'), 's-1', 'c');
    q = dropLast(q, 's-1');
    expect(q['s-1']?.items).toEqual(['a', 'b']);
    q = dropAt(q, 's-1', 0);
    expect(q['s-1']?.items).toEqual(['b']);
    q = dropLast(q, 's-1');
    expect(q['s-1']).toBeUndefined();
    expect(dropLast(noQueues, 's-1')).toBe(noQueues);
  });
});

describe('sending what was queued', () => {
  const two = enqueue(enqueue(noQueues, 's-1', 'primeiro'), 's-1', 'segundo');

  it('waits while the turn runs or an approval waits', () => {
    expect(step(two, 's-1', running(0)).send).toBeNull();
    expect(step(two, 's-1', { state: 'blocked', turns: 0 }).send).toBeNull();
  });

  it('sends the oldest when the conversation is idle, once', () => {
    const s = step(two, 's-1', idle(1));
    expect(s.send).toBe('primeiro');
    expect(s.queues['s-1']?.phase).toBe('sending');
    // Asked again before the daemon answered: nothing more goes.
    expect(step(s.queues, 's-1', idle(1)).send).toBeNull();
  });

  it('after a send, waits for that turn to start and end before the next', () => {
    let q = step(two, 's-1', idle(1)).queues;
    q = sent(q, 's-1');
    expect(q['s-1']?.items).toEqual(['segundo']);
    // Still idle, its turn.started not yet here: the next must not go, or it would cross into a steer.
    expect(step(q, 's-1', idle(1)).send).toBeNull();
    q = step(q, 's-1', running(1)).queues;
    expect(step(q, 's-1', running(1)).send).toBeNull();
    expect(step(q, 's-1', idle(2)).send).toBe('segundo');
  });

  it('notices a turn that started and ended between two looks', () => {
    let q = step(two, 's-1', idle(1)).queues;
    q = sent(q, 's-1');
    expect(step(q, 's-1', idle(2)).send).toBe('segundo');
  });

  it('is gone once the last one was sent', () => {
    const one = enqueue(noQueues, 's-1', 'só');
    const q = sent(step(one, 's-1', idle(0)).queues, 's-1');
    expect(q['s-1']).toBeUndefined();
  });

  it('stops on a failed send, keeping the message and saying why, until resumed', () => {
    let q = failed(step(two, 's-1', idle(1)).queues, 's-1', 'A mensagem não foi enviada: sem conexão');
    expect(q['s-1']).toMatchObject({ phase: 'halted', why: 'A mensagem não foi enviada: sem conexão', items: ['primeiro', 'segundo'] });
    expect(step(q, 's-1', idle(1)).send).toBeNull();
    // Queuing more does not restart it: the person decides.
    q = enqueue(q, 's-1', 'terceiro');
    expect(q['s-1']?.phase).toBe('halted');
    q = resume(q, 's-1');
    expect(q['s-1']?.why).toBe('');
    expect(step(q, 's-1', idle(1)).send).toBe('primeiro');
  });

  it('does nothing for a conversation with no queue', () => {
    expect(step(noQueues, 's-1', idle(0))).toEqual({ queues: noQueues, send: null });
  });
});
