import { describe, expect, it } from 'vitest';
import type { CommandInfo } from '../../src/protocol/generated';
import { decodeCommands } from '../../src/protocol/commands';
import { completions, expandCommand, parseInput } from '../../src/state/commands';

// The commands the person wrote (D50): listed by the daemon as the TUI finds
// them, offered in the "/" menu after the field's own, and sent as the TUI
// sends them — expanded, as a message.

const review: CommandInfo = { name: 'review', description: 'revisa o que está preparado', source: 'project', path: 'review.md', body: 'Review the staged change.' };
const fix: CommandInfo = { name: 'fix-issue', description: '', source: 'user', path: 'fix-issue.md', body: 'Fix the issue described here: $ARGUMENTS' };
// A user file cannot make /undo mean something else: the field's own wins.
const undo: CommandInfo = { name: 'undo', description: 'shadowed', source: 'user', path: 'undo.md', body: 'never sent' };
const user = [fix, review, undo];

describe('expanding a command, as config.Expand does', () => {
  it('puts what followed the name where $ARGUMENTS is, every time', () => {
    expect(expandCommand('Look at $ARGUMENTS and report on $ARGUMENTS.', ' main.go ')).toBe('Look at main.go and report on main.go.');
    expect(expandCommand('Fix: $ARGUMENTS', '')).toBe('Fix: ');
  });

  it('appends what followed after a blank line when there is no placeholder, and sends the body alone with nothing', () => {
    expect(expandCommand('Review the diff.', 'focus on tests')).toBe('Review the diff.\n\nfocus on tests');
    expect(expandCommand('Review the diff.', '   ')).toBe('Review the diff.');
  });
});

describe('a command the person wrote, typed in the field', () => {
  it('becomes a turn with its expanded text', () => {
    expect(parseInput('/review', user)).toEqual({ kind: 'turn', name: 'review', text: 'Review the staged change.' });
    expect(parseInput('/fix-issue  o login cai  ', user)).toEqual({ kind: 'turn', name: 'fix-issue', text: 'Fix the issue described here: o login cai' });
  });

  it('never shadows the field’s own command of the same name', () => {
    expect(parseInput('/undo', user)).toEqual({ kind: 'undo' });
  });

  it('is unknown when the workspace has no such command', () => {
    expect(parseInput('/review').kind).toBe('unknown');
  });
});

describe('the / menu with the person’s commands', () => {
  it('lists them after the field’s own, without the shadowed one', () => {
    const names = completions('/', user).map((c) => c.name);
    expect(names).toEqual(['help', 'init', 'plan', 'undo', 'compact', 'mode', 'loop', '!', 'fix-issue', 'review']);
  });

  it('describes each by what its file says, or by whose it is', () => {
    const [f, r] = completions('/', user).slice(-2);
    expect(r).toMatchObject({ detail: 'revisa o que está preparado', insert: '/review', ready: true, origin: 'projeto' });
    expect(f).toMatchObject({ detail: 'comando seu, sem descrição', origin: 'seu' });
  });

  it('waits for the argument of a command that takes one', () => {
    const f = completions('/fi', user)[0];
    expect(f).toMatchObject({ name: 'fix-issue', args: '[argumentos]', insert: '/fix-issue ', ready: false });
  });
});

describe('the daemon’s list of commands, read at the boundary', () => {
  it('takes the list as it came, and a null list as empty', () => {
    const list = { commands: [review], overrides: ['review'] };
    expect(decodeCommands(list)).toEqual({ ok: true, value: list });
    expect(decodeCommands({ commands: null, overrides: null })).toEqual({ ok: true, value: { commands: [], overrides: [] } });
  });

  it('says what is wrong with an answer that is not the list', () => {
    expect(decodeCommands([])).toEqual({ ok: false, reason: 'lista de comandos: a resposta não é um objeto' });
    const bad = decodeCommands({ commands: [{ name: 'x', source: 'user', path: 'x.md' }], overrides: [] });
    expect(!bad.ok && bad.reason).toMatch(/^lista de comandos: commands\[0\]/);
  });
});
