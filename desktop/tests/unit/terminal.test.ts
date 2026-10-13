import { describe, expect, it } from 'vitest';
import {
  MAX_LINES,
  cleanOutput,
  emptyTranscript,
  feed,
  pendingResult,
  resultMessage,
  stripAnsi,
  tailLines,
} from '../../src/state/terminal';

// What the embedded terminal hands back to the conversation: the PTY's bytes
// read as text, the command the shell said it ran and how it ended, cut to a
// size a message can carry — and never what was typed at a password prompt.

const b64 = (s: string) => btoa(String.fromCharCode(...new TextEncoder().encode(s)));
const C = (cmd?: string) => `\x1b]133;C${cmd === undefined ? '' : `;${b64(cmd)}`}\x07`;
const D = (code?: number, cmd?: string) => `\x1b]133;D${code === undefined ? '' : `;${code};${b64(cmd ?? '')}`}\x07`;

describe('the terminal output, as text', () => {
  it('strips colours, cursor moves, titles and other escapes', () => {
    expect(stripAnsi('\x1b[1;32mok\x1b[0m done')).toBe('ok done');
    expect(stripAnsi('\x1b]0;title\x07a\x1b]2;t\x1b\\b')).toBe('ab');
    expect(stripAnsi('x\x1b[?2004hy\x1b[2Kz\x1b(Bw\x1b=v')).toBe('xyzwv');
    expect(stripAnsi('bell\x07 and nul\x00')).toBe('bell and nul');
  });

  it('keeps what a carriage return left on the line, as a terminal shows it', () => {
    expect(cleanOutput('10%\r50%\r100%\r\nnext\r\n')).toBe('100%\nnext');
    expect(cleanOutput('abc\b\bX\n')).toBe('aX');
  });

  it('drops the lines of a password prompt', () => {
    const out = cleanOutput('[sudo] password for ana: \r\nPassword:\r\nSenha:\r\nreal output\r\n');
    expect(out).toBe('real output');
    expect(cleanOutput('Sorry, try again.\n[sudo] senha para ana:\nok')).toBe('Sorry, try again.\nok');
  });

  it('keeps the last lines and says how many there were', () => {
    const many = Array.from({ length: 450 }, (_, i) => `line ${i + 1}`).join('\n');
    const t = tailLines(many);
    expect(MAX_LINES).toBe(200);
    expect(t.truncated).toBe(true);
    expect(t.total).toBe(450);
    expect(t.text.split('\n')).toHaveLength(200);
    expect(t.text.startsWith('line 251')).toBe(true);
    expect(tailLines('a\nb')).toEqual({ text: 'a\nb', truncated: false, total: 2 });
  });

  it('caps a few enormous lines by size too', () => {
    const t = tailLines('x'.repeat(50_000));
    expect(t.truncated).toBe(true);
    expect(t.text.length).toBeLessThanOrEqual(16_000);
  });
});

describe('the shell marks, folded', () => {
  it('reads a command that ran, its code and only its output', () => {
    let t = emptyTranscript;
    t = feed(t, `${D()}% `); // the first prompt: integration works, nothing ran
    expect(t.prompts).toBe(1);
    expect(t.last).toBeNull();
    t = feed(t, `npm test\r\n${C('npm test')}\x1b[31mfail\x1b[0m\r\n`);
    expect(t.running).toEqual({ command: 'npm test' });
    t = feed(t, `${D(1, 'npm test')}% `);
    expect(t.running).toBeNull();
    expect(t.last).toEqual({ command: 'npm test', code: 1, output: '\x1b[31mfail\x1b[0m\r\n' });
  });

  it('reads a mark split across two chunks', () => {
    let t = feed(feed(emptyTranscript, D()), `ls\r\n${C('ls')}a\r\n\x1b]133;D;0;`);
    expect(t.last).toBeNull();
    t = feed(t, `${b64('ls')}\x07% `);
    expect(t.last).toEqual({ command: 'ls', code: 0, output: 'a\r\n' });
  });

  it('without a start mark, drops the prompt line from what it reads as output', () => {
    let t = feed(emptyTranscript, `${D()}$ `);
    t = feed(t, `echo hi\r\nhi\r\n${D(0, 'echo hi')}$ `);
    expect(t.last?.output).toBe('hi\r\n');
  });

  it('an empty prompt mark is not a command', () => {
    let t = feed(emptyTranscript, `${D()}$ `);
    t = feed(t, `\r\n${D()}$ `);
    expect(t.last).toBeNull();
    expect(t.prompts).toBe(2);
  });

  it('decodes a command in UTF-8', () => {
    let t = feed(emptyTranscript, D());
    t = feed(t, `${C('echo ção')}ção\r\n${D(0, 'echo ção')}`);
    expect(t.last?.command).toBe('echo ção');
  });
});

describe('the message to the agent', () => {
  it('says the command, the code and the output, in a fence', () => {
    const m = resultMessage({ command: 'npm test', code: 1, output: 'FAIL a.test\r\n' });
    expect(m).toBe('Rodei no terminal: `npm test` — saída (código 1):\n```\nFAIL a.test\n```');
  });

  it('says when the output was cut', () => {
    const output = Array.from({ length: 300 }, (_, i) => `l${i}`).join('\n');
    const m = resultMessage({ command: 'ls', code: 0, output });
    expect(m).toContain('(mostrando as últimas 200 de 300 linhas)');
    expect(m).not.toContain('\nl99\n');
    expect(m).toContain('\nl299\n');
  });

  it('says there was no output, and when the code is unknown', () => {
    expect(resultMessage({ command: 'true', code: 0, output: '' })).toBe('Rodei no terminal: `true` — sem saída (código 0).');
    expect(resultMessage({ command: 'make', code: null, output: 'x' })).toContain('saída (ainda rodando, ou sem código de saída):');
  });

  it('never carries the password prompt, and survives backticks', () => {
    const m = resultMessage({ command: 'sudo ls `pwd`', code: 0, output: '[sudo] password for ana: \r\nfile\r\n```\r\n' });
    expect(m).not.toMatch(/password for/);
    expect(m.startsWith('Rodei no terminal: `` sudo ls `pwd` `` — saída (código 0):\n````\n')).toBe(true);
    expect(m.endsWith('\n````')).toBe(true);
  });

  it('puts a command of several lines in its own block', () => {
    const m = resultMessage({ command: 'cat <<EOF\nx\nEOF', code: 0, output: 'x' });
    expect(m).toBe('Rodei no terminal:\n```sh\ncat <<EOF\nx\nEOF\n```\nsaída (código 0):\n```\nx\n```');
  });

  it('offers what is there when asked by hand', () => {
    let t = feed(emptyTranscript, D());
    expect(pendingResult(t, 'npm i')).toEqual({ command: 'npm i', code: null, output: '' });
    t = feed(t, `${C('sudo npm i')}added 3\r\n`);
    expect(pendingResult(t, 'npm i')).toEqual({ command: 'sudo npm i', code: null, output: 'added 3\r\n' });
    t = feed(t, `${D(0, 'sudo npm i')}% `);
    expect(pendingResult(t, 'npm i')).toEqual({ command: 'sudo npm i', code: 0, output: 'added 3\r\n' });
  });
});
