import { describe, expect, it } from 'vitest';
import tui from '../../../internal/tui/commands.go?raw';
import { COMMANDS, completions, INIT_PROMPT, parseInput, planSummary, replanPrompt } from '../../src/state/commands';
import { HELP_KEYS } from '../../src/state/help';

// The "/" menu and /help (D48): every command the field knows is offered as it
// is typed, and /init and /plan send the turns the TUI sends.

describe('the / menu', () => {
  it('opens on a bare slash at the start of the field, with every command and !', () => {
    const all = completions('/');
    expect(all.map((c) => c.name)).toEqual(['help', 'init', 'plan', 'undo', 'compact', 'mode', '!']);
    expect(all).toHaveLength(COMMANDS.length);
  });

  it('filters by what follows the slash, without case', () => {
    expect(completions('/u').map((c) => c.name)).toEqual(['undo']);
    expect(completions('/C').map((c) => c.name)).toEqual(['compact']);
    expect(completions('/xyz')).toEqual([]);
  });

  it('closes once an argument begins, and stays shut for anything that is not a leading slash', () => {
    expect(completions('/mode ')).toEqual([]);
    expect(completions(' /undo')).toEqual([]);
    expect(completions('use a/b')).toEqual([]);
    expect(completions('')).toEqual([]);
  });

  it('says what accepting writes, and whether it can be sent as it is', () => {
    const by = Object.fromEntries(COMMANDS.map((c) => [c.name, c]));
    expect(by.undo).toMatchObject({ insert: '/undo', ready: true });
    expect(by.plan).toMatchObject({ insert: '/plan', ready: true });
    expect(by.mode).toMatchObject({ insert: '/mode ', ready: false });
    expect(by['!']).toMatchObject({ insert: '!', ready: false });
  });
});

describe('/help, /init and /plan', () => {
  it('reads /help as the help, not as a message', () => {
    expect(parseInput('/help')).toEqual({ kind: 'help' });
    expect(parseInput('  /help  ')).toEqual({ kind: 'help' });
  });

  it('sends /init as the turn the TUI sends, word for word', () => {
    const literal = /const InitPrompt = `([^`]*)`/.exec(tui)?.[1];
    expect(literal, 'InitPrompt not found in internal/tui/commands.go').toBeTruthy();
    expect(INIT_PROMPT).toBe(literal);
    expect(parseInput('/init')).toEqual({ kind: 'turn', name: 'init', text: INIT_PROMPT });
  });

  it('shows the plan for a bare /plan, and asks for a new one with text after it', () => {
    expect(parseInput('/plan')).toEqual({ kind: 'plan' });
    expect(parseInput('/plan  use sqlite ')).toEqual({ kind: 'turn', name: 'plan', text: 'Revise the plan to account for this: use sqlite' });
    expect(replanPrompt('')).toBe('Re-examine the plan and revise it if it no longer fits what you have learned.');
  });

  it('counts the plan for its heading, and says when there is none', () => {
    expect(planSummary([])).toBe('Ainda não há plano nesta conversa.');
    expect(
      planSummary([
        { id: 1, text: 'a', status: 'done' },
        { id: 2, text: 'b', status: 'active' },
        { id: 3, text: 'c', status: 'blocked', blocked: 'sem rede' },
      ]),
    ).toBe('1 de 3 feito · 1 em andamento · 1 bloqueado');
  });
});

describe('the keys the help lists', () => {
  it('names the window keys useWindowKeys answers, and the field keys', () => {
    const keys = HELP_KEYS.flatMap((g) => g.keys.map((k) => k.key));
    for (const k of ['⌘K', '⌘N', '⌘0', '⌘1–9', '⌘↵', '⇧⇥', '↵', '⇧↵', '↑ ↓', '⌥↵', '⌃X', '/', '?', '1–4']) expect(keys).toContain(k);
  });
});
