import { describe, expect, it } from 'vitest';
import { DECISIONS, conversation, decision, id, text, workspace } from '../../src/main/inputs';

// What the renderer hands the main process, checked before it reaches the
// daemon: a bad argument is a refusal with the reason, never a throw.

function refused(got: { ok: boolean; refusal?: { code: string; message: string } }): string {
  expect(got.ok).toBe(false);
  expect(got.refusal?.code).toBe('invalid_input');
  return got.refusal?.message ?? '';
}

describe('an argument from the window', () => {
  it('takes an id that says something, and refuses one that does not', () => {
    expect(id('s-1', 'O id da sessão')).toEqual({ ok: true, value: 's-1' });
    expect(refused(id('', 'O id da sessão'))).toBe('O id da sessão precisa ser um texto não vazio, e veio “”.');
    refused(id('   ', 'O id da sessão'));
    expect(refused(id(42, 'O id da sessão'))).toContain('number 42');
    expect(refused(id(undefined, 'O id da sessão'))).toContain('veio nada');
  });

  it('takes text a person wrote, and refuses an empty message', () => {
    expect(text('diga oi')).toEqual({ ok: true, value: 'diga oi' });
    refused(text(''));
    refused(text('\n  '));
    refused(text({ text: 'oi' }));
  });

  it('takes an absolute folder, and refuses a relative one, naming it', () => {
    expect(workspace('/Users/ana/projeto')).toEqual({ ok: true, value: '/Users/ana/projeto' });
    expect(refused(workspace('projeto'))).toBe('A pasta do projeto precisa ser um caminho absoluto, e é “projeto”.');
    refused(workspace(''));
    refused(workspace(null));
  });

  it('takes the protocol’s five answers to an approval, and nothing else', () => {
    expect(DECISIONS).toEqual(['allow', 'allow_session', 'allow_project', 'allow_always', 'deny']);
    for (const d of DECISIONS) expect(decision(d)).toEqual({ ok: true, value: d });
    expect(refused(decision('yes'))).toContain('allow, allow_session, allow_project, allow_always, deny');
    refused(decision(1));
  });

  it('takes the id and folder of a conversation to continue, and only those', () => {
    const c = { id: 'c-1', workspace: '/w', title: 'conserte o parser', live: false };
    expect(conversation(c)).toEqual({ ok: true, value: { id: 'c-1', workspace: '/w' } });
    refused(conversation(null));
    refused(conversation('c-1'));
    expect(refused(conversation({ id: 'c-1', workspace: 'w' }))).toContain('absoluto');
    expect(refused(conversation({ workspace: '/w' }))).toContain('O id da conversa');
  });

  it('cuts a long value short in the reason', () => {
    expect(refused(workspace('x'.repeat(500))).length).toBeLessThan(200);
  });
});
