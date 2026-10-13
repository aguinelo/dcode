import { describe, expect, it } from 'vitest';
import { integration, launch, pickShell, typed } from '../../src/main/shell';

// Which shell the embedded terminal runs, how it is told to mark its prompts,
// and how a proposed command is typed into it without being run.

describe('the shell', () => {
  it('is the person’s own, else the platform’s', () => {
    expect(pickShell({ SHELL: '/opt/homebrew/bin/zsh' }, 'darwin')).toEqual({ file: '/opt/homebrew/bin/zsh', kind: 'zsh' });
    expect(pickShell({ SHELL: '/bin/bash' }, 'linux')).toEqual({ file: '/bin/bash', kind: 'bash' });
    expect(pickShell({ SHELL: '/usr/bin/fish' }, 'linux')).toEqual({ file: '/usr/bin/fish', kind: 'other' });
    expect(pickShell({}, 'darwin')).toEqual({ file: '/bin/zsh', kind: 'zsh' });
    expect(pickShell({ SHELL: 'zsh' }, 'linux')).toEqual({ file: '/bin/bash', kind: 'bash' });
  });

  it('starts zsh as a login shell through the app’s own ZDOTDIR, keeping the person’s', () => {
    const l = launch({ file: '/bin/zsh', kind: 'zsh' }, '/data/shell', { HOME: '/Users/ana', PATH: '/usr/bin', ELECTRON_RUN_AS_NODE: '1' });
    expect(l.file).toBe('/bin/zsh');
    expect(l.args).toEqual(['-l']);
    expect(l.env.ZDOTDIR).toBe('/data/shell/zsh');
    expect(l.env.DCODE_USER_ZDOTDIR).toBe('/Users/ana');
    expect(l.env.TERM).toBe('xterm-256color');
    expect(l.env.ELECTRON_RUN_AS_NODE).toBeUndefined();
    expect(launch({ file: '/bin/zsh', kind: 'zsh' }, '/d', { HOME: '/h', ZDOTDIR: '/h/.config/zsh' }).env.DCODE_USER_ZDOTDIR).toBe('/h/.config/zsh');
  });

  it('starts bash on the app’s rc file, and another shell as it is', () => {
    expect(launch({ file: '/bin/bash', kind: 'bash' }, '/d', {}).args).toEqual(['--rcfile', '/d/bash/rc', '-i']);
    expect(launch({ file: '/usr/bin/fish', kind: 'other' }, '/d', {}).args).toEqual(['-l']);
  });

  it('marks the prompt, the start of a command and its code, after the person’s rc', () => {
    const zsh = integration('zsh');
    const rc = zsh.find((f) => f.name === 'zsh/.zshrc')?.content ?? '';
    expect(rc.indexOf('.zshrc"')).toBeLessThan(rc.indexOf('precmd_functions'));
    expect(rc).toContain('133;D');
    expect(rc).toContain('133;C');
    expect(zsh.map((f) => f.name).sort()).toEqual(['zsh/.zlogin', 'zsh/.zprofile', 'zsh/.zshenv', 'zsh/.zshrc']);
    const bash = integration('bash');
    expect(bash.map((f) => f.name)).toEqual(['bash/rc']);
    expect(bash[0]?.content).toContain('PROMPT_COMMAND');
    expect(integration('other')).toEqual([]);
  });
});

describe('a proposed command, typed', () => {
  it('goes into zsh as a paste, so even several lines wait for Enter', () => {
    expect(typed('zsh', 'rm -rf x\nls')).toEqual({ ok: true, value: '\x1b[200~rm -rf x\nls\x1b[201~' });
  });

  it('goes into another shell as it is, and only on one line', () => {
    expect(typed('bash', 'ls -la')).toEqual({ ok: true, value: 'ls -la' });
    const multi = typed('bash', 'a\nb');
    expect(multi.ok).toBe(false);
    if (!multi.ok) expect(multi.refusal.message).toContain('várias linhas');
  });

  it('is never typed with what would run it or escape the paste', () => {
    for (const bad of ['ls\r', 'ls\x1b[201~; rm -rf ~', 'a\x03', '']) {
      expect(typed('zsh', bad).ok).toBe(false);
    }
  });
});
