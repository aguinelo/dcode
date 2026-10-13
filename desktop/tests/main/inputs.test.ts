import { describe, expect, it } from 'vitest';
import {
  DECISIONS,
  MODES,
  SANDBOXES,
  clip,
  command,
  NAME_LIMIT,
  conversation,
  conversationName,
  decision,
  gitWorkspaceOf,
  id,
  keys,
  mode,
  model,
  position,
  recordedId,
  repoPath,
  sessionOptions,
  size,
  text,
  workspace,
  workspaceOf,
} from '../../src/main/inputs';

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

  it('takes a profile or model name, and refuses an empty or spaced one', () => {
    expect(model('qwen-local')).toEqual({ ok: true, value: 'qwen-local' });
    expect(refused(model(''))).toContain('modelo');
    expect(refused(model('qwen local'))).toContain('sem espaços');
    refused(model(7));
  });

  it('takes the three behavioural modes the daemon knows, and nothing else', () => {
    expect(MODES).toEqual(['plan', 'assist', 'auto']);
    for (const m of MODES) expect(mode(m)).toEqual({ ok: true, value: m });
    expect(refused(mode('full-access'))).toContain('plan, assist, auto');
    refused(mode(''));
    refused(mode(undefined));
  });

  it('takes the options of a new session, each only when given, and refuses a bad one', () => {
    expect(SANDBOXES).toEqual(['read-only', 'workspace-write', 'full-access']);
    expect(sessionOptions(undefined)).toEqual({ ok: true, value: {} });
    expect(sessionOptions({})).toEqual({ ok: true, value: {} });
    expect(sessionOptions({ model: 'qwen-local', sandboxMode: 'full-access' })).toEqual({
      ok: true,
      value: { model: 'qwen-local', sandboxMode: 'full-access' },
    });
    expect(refused(sessionOptions({ sandboxMode: 'yolo' }))).toContain('read-only, workspace-write, full-access');
    expect(refused(sessionOptions({ model: 'qwen local' }))).toContain('sem espaços');
    refused(sessionOptions('qwen'));
  });

  it('cuts a long value short in the reason', () => {
    expect(refused(workspace('x'.repeat(500))).length).toBeLessThan(200);
  });

  it('takes a command to type into a terminal, and refuses an empty or huge one', () => {
    expect(command('sudo npm i')).toEqual({ ok: true, value: 'sudo npm i' });
    expect(refused(command(''))).toContain('comando');
    refused(command(3));
    refused(command('x'.repeat(20_000)));
  });

  it('takes keys for a terminal as they came, and a size in cells', () => {
    expect(keys('\r')).toEqual({ ok: true, value: '\r' });
    expect(keys('\x1b[A')).toEqual({ ok: true, value: '\x1b[A' });
    refused(keys(''));
    refused(keys({}));
    refused(keys('x'.repeat(70_000)));
    expect(size(80, 24)).toEqual({ ok: true, value: { cols: 80, rows: 24 } });
    refused(size(0, 24));
    refused(size(80.5, 24));
    refused(size(80, 5000));
    refused(size('80', 24));
  });

  it('takes text to copy, and refuses nothing to copy', () => {
    expect(clip('ls -la')).toEqual({ ok: true, value: 'ls -la' });
    refused(clip(''));
    refused(clip(null));
  });
});

describe('a recorded conversation to read (D42)', () => {
  it('takes an id the daemon could name a record by, and refuses one that is a path', () => {
    expect(recordedId('0199a1b2c3d4e5f6a7b')).toEqual({ ok: true, value: '0199a1b2c3d4e5f6a7b' });
    expect(recordedId('s-1_a')).toEqual({ ok: true, value: 's-1_a' });
    for (const bad of ['', '..', '../secret', 'a/b', 'a\\b', 'a.jsonl', '%2e%2e', 'a b', 'x'.repeat(129), 7, null]) {
      expect(refused(recordedId(bad))).toMatch(/O id da conversa/);
    }
  });

  it('takes a position from 1, and refuses anything else', () => {
    expect(position(1)).toEqual({ ok: true, value: 1 });
    expect(position(5001)).toEqual({ ok: true, value: 5001 });
    for (const bad of [0, -1, 1.5, Number.NaN, Number.MAX_SAFE_INTEGER + 1, '1', undefined]) refused(position(bad));
  });
});

describe('a conversation’s name (D43)', () => {
  it('takes a name trimmed, and an empty one, which gives the derived title back', () => {
    expect(conversationName('  Migração do banco  ')).toEqual({ ok: true, value: 'Migração do banco' });
    expect(conversationName('')).toEqual({ ok: true, value: '' });
    expect(conversationName('   ')).toEqual({ ok: true, value: '' });
  });

  it('counts characters as the daemon does, and refuses what it would refuse rather than cutting it', () => {
    expect(NAME_LIMIT).toBe(120);
    expect(conversationName('é'.repeat(120))).toEqual({ ok: true, value: 'é'.repeat(120) });
    expect(conversationName('🙂'.repeat(120)).ok).toBe(true);
    expect(refused(conversationName('x'.repeat(121)))).toBe('O nome tem 121 caracteres, e o daemon guarda até 120.');
  });

  it('refuses what is not text', () => {
    for (const bad of [undefined, null, 7, ['a'], { name: 'a' }]) expect(refused(conversationName(bad))).toMatch(/^O nome da conversa precisa ser um texto/);
  });
});

describe('the folder a session’s terminal opens in', () => {
  const list = [
    { id: 's-1', workspace: '/work/app' },
    { id: 's-2', workspace: '/work/other' },
  ];

  it('is the workspace the daemon listed for the session, never one the window names', () => {
    expect(workspaceOf(list, 's-2')).toEqual({ ok: true, value: '/work/other' });
    expect(refused(workspaceOf(list, 's-9'))).toContain('s-9');
    refused(workspaceOf(null, 's-1'));
  });
});

describe('the folder whose git a conversation’s panel reads (D41)', () => {
  const list = [
    { id: 's-1', workspace: '/work/app' },
    { id: 'old', workspace: '/work/ended' },
  ];

  it('is the workspace the daemon listed for the conversation, live or ended, never one the window names', () => {
    expect(gitWorkspaceOf(list, 'old')).toEqual({ ok: true, value: '/work/ended' });
    expect(refused(gitWorkspaceOf(list, 's-9'))).toContain('s-9');
    refused(gitWorkspaceOf(null, 's-1'));
    refused(gitWorkspaceOf([{ id: 'rel', workspace: 'work/app' }], 'rel'));
  });

  it('takes a file as the list names it, and refuses nothing, a NUL or a runaway path', () => {
    expect(repoPath('internal/vcs/tree.go')).toEqual({ ok: true, value: 'internal/vcs/tree.go' });
    refused(repoPath(''));
    refused(repoPath(42));
    refused(repoPath('a\0b'));
    refused(repoPath('a/'.repeat(3000)));
  });
});
