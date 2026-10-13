import { describe, expect, it } from 'vitest';
import { answerForKey, choicesFor } from '../../src/state/approval';

// The answers an approval offers, as the TUI offers them (D39): a network
// crossing can be answered for the project or for good, and not for the
// session; any other crossing once or for the session. Allowing is always an
// explicit key, and ↵ and esc deny (D22).

const network = { boundary_crossed: 'network' };
const write = { boundary_crossed: 'filesystem_write' };

describe('the answers to an approval', () => {
  it('offers once, this project, always and deny on a network crossing — and not this session', () => {
    expect(choicesFor(network).map((c) => [c.key, c.decision, c.label])).toEqual([
      ['1', 'allow', 'Permitir uma vez'],
      ['2', 'allow_project', 'Permitir neste projeto'],
      ['3', 'allow_always', 'Permitir sempre'],
      ['4', 'deny', 'Negar'],
    ]);
  });

  it('offers once, this session and deny on any other crossing', () => {
    expect(choicesFor(write).map((c) => [c.key, c.decision])).toEqual([
      ['1', 'allow'],
      ['2', 'allow_session'],
      ['3', 'deny'],
    ]);
  });

  it('marks the answers that outlive the session as standing', () => {
    expect(choicesFor(network).filter((c) => c.standing).map((c) => c.decision)).toEqual(['allow_project', 'allow_always']);
    expect(choicesFor(write).some((c) => c.standing)).toBe(false);
  });

  it('gives each key its own answer for the crossing, and ↵ and esc deny', () => {
    expect(answerForKey('2', network)).toBe('allow_project');
    expect(answerForKey('3', network)).toBe('allow_always');
    expect(answerForKey('4', network)).toBe('deny');
    expect(answerForKey('2', write)).toBe('allow_session');
    expect(answerForKey('3', write)).toBe('deny');
    expect(answerForKey('4', write)).toBeNull();
    expect(answerForKey('Enter', network)).toBe('deny');
    expect(answerForKey('Escape', write)).toBe('deny');
    expect(answerForKey('a', write)).toBeNull();
  });
});
