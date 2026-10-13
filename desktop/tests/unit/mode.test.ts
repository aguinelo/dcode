import { describe, expect, it } from 'vitest';
import { MODES, modeChip, nextMode } from '../../src/state/mode';

// The session's behavioural mode as the window names and switches it (D36):
// the three the daemon knows, in the TUI's order, and a pair that is none of
// them named as such rather than as the nearest one.

describe('the mode of a session', () => {
  it('offers the three the daemon knows, in the TUI’s order, in the window’s words', () => {
    expect(MODES.map((m) => [m.mode, m.label])).toEqual([
      ['plan', 'Plano'],
      ['assist', 'Assistir'],
      ['auto', 'Auto'],
    ]);
  });

  it('cycles plan → assist → auto → plan, and enters at plan from a pair with no name, as ⇧⇥ in the TUI', () => {
    expect(nextMode('plan')).toBe('assist');
    expect(nextMode('assist')).toBe('auto');
    expect(nextMode('auto')).toBe('plan');
    expect(nextMode('')).toBe('plan');
  });

  it('draws the chip as the bar did, and auto in the warning tone', () => {
    expect(modeChip('assist', 'workspace-write')).toMatchObject({ text: 'workspace-write · assist', tone: 'quiet', named: true });
    expect(modeChip('plan', 'read-only')).toMatchObject({ text: 'read-only · plan', tone: 'quiet' });
    expect(modeChip('auto', 'full-access')).toMatchObject({ text: 'full-access · auto', tone: 'warn' });
  });

  it('names a pair that is none of the three as custom, never as the nearest mode', () => {
    const chip = modeChip('', 'read-only');
    expect(chip).toMatchObject({ text: 'read-only · personalizado', tone: 'quiet', named: false });
    expect(chip?.title).toContain('personalizado (sandbox · política)');
  });

  it('draws nothing before the session says what it runs under', () => {
    expect(modeChip('', '')).toBeNull();
  });
});
