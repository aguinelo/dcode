import { describe, expect, it } from 'vitest';
import { flowBlocks } from '../../src/state/flow';
import { foldHistory } from '../../src/state/history';
import type { ToolCall } from '../../src/state/session';
import { OUTPUT_PREVIEW_LINES, stripAnsi, toolDetail } from '../../src/state/tooldetail';
import { done, log } from './helpers';

function call(name: string, result: Partial<NonNullable<ToolCall['result']>> | null, typed = false): ToolCall {
  return {
    id: 'c1',
    name,
    input: { path: 'a.go' },
    typed,
    status: result ? (result.ok === false ? 'failed' : 'ok') : 'running',
    requestedAt: '2026-09-28T17:00:00Z',
    result: result ? { tool_call_id: 'c1', ok: true, output: '', truncated: false, ...result } : null,
    progress: null,
  };
}

const lines = (n: number) => Array.from({ length: n }, (_, i) => `line ${i + 1}`).join('\n');

describe('what a tool call shows when it is opened', () => {
  it('has nothing to open while it runs, nor when it said nothing', () => {
    expect(toolDetail(call('read', null))).toBeNull();
    expect(toolDetail(call('glob', { output: '' }))).toBeNull();
  });

  it('leaves a typed command alone: it is already shown open', () => {
    expect(toolDetail(call('bash', { output: 'hi' }, true))).toBeNull();
  });

  it('shows the output with its colour codes taken out', () => {
    const d = toolDetail(call('bash', { output: '\u001b[32mok\u001b[0m  pkg\n\u001b]0;title\u0007done' }));
    expect(d?.output?.text).toBe('ok  pkg\ndone');
    expect(d?.diff).toBeNull();
  });

  it('keeps only what a carriage return left on screen', () => {
    expect(stripAnsi('10%\r50%\r100%\nnext')).toBe('100%\nnext');
    expect(stripAnsi('a\r\nb')).toBe('a\nb');
  });

  it('cuts a long output to a preview and says how much there is', () => {
    const d = toolDetail(call('read', { output: lines(OUTPUT_PREVIEW_LINES + 10) }));
    expect(d?.output?.preview.split('\n')).toHaveLength(OUTPUT_PREVIEW_LINES);
    expect(d?.output?.totalLines).toBe(OUTPUT_PREVIEW_LINES + 10);
    expect(d?.output?.long).toBe(true);
    const short = toolDetail(call('read', { output: lines(3) }));
    expect(short?.output?.long).toBe(false);
    expect(short?.output?.preview).toBe(short?.output?.text);
  });

  it('says when the tool itself cut what it returned', () => {
    expect(toolDetail(call('read', { output: 'x', truncated: true }))?.output?.cutByTool).toBe(true);
  });

  it('shows a failure whole, never a preview of it', () => {
    const d = toolDetail(call('bash', { ok: false, output: lines(OUTPUT_PREVIEW_LINES * 2) }));
    expect(d?.failed).toBe(true);
    expect(d?.output?.long).toBe(false);
    expect(d?.output?.preview).toBe(d?.output?.text);
  });

  it('shows a change as its diff, coloured by line, and not the output beside it', () => {
    const diff = '@@ -1,2 +1,2 @@ a.go\n package a\n-var x = 1\n+var x = 2';
    const d = toolDetail(call('edit', { output: 'edited a.go\n' + diff, diff, added: 1, removed: 1 }));
    expect(d?.output).toBeNull();
    expect(d?.diff?.lines.map((l) => l.kind)).toEqual(['hunk', 'ctx', 'del', 'add']);
    expect(d?.diff?.cutAt).toBeNull();
  });

  it('reads the daemon’s cut mark as a cut, not as a line of the file', () => {
    const diff = '@@ -1,1 +1,1 @@\n-a\n+b\n⋯ diff truncated at 400 lines';
    const d = toolDetail(call('write', { diff }));
    expect(d?.diff?.lines.map((l) => l.kind)).toEqual(['hunk', 'del', 'add']);
    expect(d?.diff?.cutAt).toBe(400);
  });

  it('shows the error of a failed change even when a diff came with it', () => {
    const d = toolDetail(call('edit', { ok: false, output: 'old_string not found', diff: '@@ -1 +1 @@\n-a\n+b' }));
    expect(d?.output?.text).toBe('old_string not found');
    expect(d?.diff).not.toBeNull();
  });
});

describe('the detail, folded from the events', () => {
  const events = () =>
    log()
      .add('turn.started', { turn_id: 't', text: 'troca x' })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'edit', input: { path: 'a.go' } })
      .add('tool.completed', done('c1', { output: 'edited', diff: '@@ -1 +1 @@\n-a\n+b', added: 1, removed: 1 }))
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c2', name: 'bash', input: { command: 'go test' } })
      .add('tool.completed', done('c2', { ok: false, output: '\u001b[31mFAIL\u001b[0m a', has_exit: true, exit_code: 1 }));

  const details = (entries: Parameters<typeof flowBlocks>[0]) =>
    flowBlocks(entries)
      .flatMap((b) => (b.kind === 'tools' ? b.calls : []))
      .map((c) => toolDetail(c));

  it('reaches a live session’s tool lines', () => {
    const [edit, bash] = details(events().fold().entries);
    expect(edit?.diff?.lines.map((l) => l.kind)).toEqual(['hunk', 'del', 'add']);
    expect(bash?.failed).toBe(true);
    expect(bash?.output?.text).toBe('FAIL a');
  });

  it('reaches a recorded conversation’s tool lines the same way', () => {
    const [edit, bash] = details(foldHistory('s-1', events().events).entries);
    expect(edit?.diff?.lines.map((l) => l.kind)).toEqual(['hunk', 'del', 'add']);
    expect(bash?.output?.text).toBe('FAIL a');
  });
});
