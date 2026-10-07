import { describe, expect, it } from 'vitest';
import { activity, delegation, flowBlocks, sealOf, toolLine, toolSummary } from '../../src/state/flow';
import type { ToolCall } from '../../src/state/session';
import { done, log } from './helpers';

function call(name: string, input: unknown, result: Partial<ToolCall['result']> | null, status?: ToolCall['status']): ToolCall {
  return {
    id: `${name}-1`,
    name,
    input,
    typed: false,
    status: status ?? (result ? (result.ok === false ? 'failed' : 'ok') : 'running'),
    requestedAt: '2026-09-28T17:00:00Z',
    result: result ? { tool_call_id: `${name}-1`, ok: true, output: '', truncated: false, ...result } : null,
    progress: null,
  };
}

describe('a tool line', () => {
  it('summarises from the fields the tool reported', () => {
    expect(toolSummary(call('grep', { pattern: 'x' }, { lines: 11, files: 4 }))).toBe('11 matches · 4 arquivos');
    expect(toolSummary(call('grep', { pattern: 'x' }, { lines: 1, files: 1 }))).toBe('1 match · 1 arquivo');
    expect(toolSummary(call('grep', { pattern: 'x' }, {}))).toBe('nenhum match');
    expect(toolSummary(call('read', { path: 'a' }, { lines: 96 }))).toBe('96 linhas');
    expect(toolSummary(call('write', { path: 'a' }, { added: 54 }))).toBe('criado, 54 linhas');
    expect(toolSummary(call('write', { path: 'a' }, { added: 5, removed: 2 }))).toBe('+5 −2');
    expect(toolSummary(call('edit', { path: 'a' }, { added: 3, removed: 1 }))).toBe('+3 −1');
    expect(toolSummary(call('bash', { command: 'go test' }, { has_exit: true, exit_code: 0 }))).toBe('exit 0');
    expect(toolSummary(call('explore', { path: 'a/' }, { files: 9 }))).toBe('leu 9');
  });

  it('never reads a number back out of the output text', () => {
    // The output says 12 lines; the tool reported nothing, so nothing is counted.
    expect(toolSummary(call('read', { path: 'a' }, { output: '12 lines\nmore' }))).toBe('12 lines');
    expect(toolSummary(call('glob', { pattern: '*.go' }, { output: '7 files' }))).toBe('0 arquivos');
  });

  it('names a failure by its first line, and a running call with an ellipsis', () => {
    const failed = toolLine(call('bash', { command: 'x' }, { ok: false, output: 'exit 1\n--- FAIL' }));
    expect(failed).toMatchObject({ glyph: '⊘', glyphTone: 'err', summary: 'exit 1', summaryTone: 'err' });
    expect(toolLine(call('grep', { pattern: '\\.Save\\(', path: 'internal' }, null))).toMatchObject({
      glyph: '●',
      target: '\\.Save\\(',
      summary: '…',
    });
  });
});

describe('the flow', () => {
  const explore = (id: string, dir: string) =>
    ({ turn_id: 't', tool_call_id: id, name: 'explore', input: { task: 't', path: dir, owns: [dir] } }) as const;

  it('groups adjacent calls, and two or more explore calls into one delegation', () => {
    const v = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'r', name: 'read', input: { path: 'a' } })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'w', name: 'write', input: { path: 'b' } })
      .add('message.delta', { turn_id: 't', text: 'Vou repartir.' })
      .add('tool.requested', explore('e1', 'internal/alpha/'))
      .add('tool.requested', explore('e2', 'internal/bravo/'))
      .add('message.delta', { turn_id: 't', text: 'E um só:' })
      .add('tool.requested', explore('e3', 'internal/tui/'))
      .fold();
    expect(flowBlocks(v.entries).map((b) => (b.kind === 'entry' ? b.entry.kind : `${b.kind}:${b.calls.length}`))).toEqual([
      'tools:2',
      'model',
      'delegation:2',
      'model',
      'tools:1',
    ]);
  });

  it('draws a call waiting on a person as its card, and after the card once answered', () => {
    const l = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c3', name: 'bash', input: { command: 'npm i' } })
      .add('tool.approval_required', {
        approval_id: 'a',
        turn_id: 't',
        tool_call_id: 'c3',
        tool: 'bash',
        boundary_crossed: 'network',
        expires_at: '0001-01-01T00:00:00Z',
      });
    expect(flowBlocks(l.fold().entries).map((b) => b.kind)).toEqual(['entry']);
    const answered = l.add('tool.approval_resolved', { approval_id: 'a', decision: 'allow' }).add('tool.completed', done('c3')).fold();
    expect(flowBlocks(answered.entries).map((b) => (b.kind === 'entry' ? b.entry.kind : b.kind))).toEqual(['approval', 'tools']);
  });
});

describe('a delegation', () => {
  const kids = [
    call('explore', { path: 'internal/alpha/', owns: ['internal/alpha/'] }, { files: 9 }),
    call('explore', { path: 'internal/bravo/', owns: ['internal/bravo/'] }, { ok: false, output: 'the delegated turn failed: context deadline exceeded' }),
    call('explore', { path: 'internal/store/', owns: ['internal/store/'] }, null),
    call('explore', { path: 'internal/tui/', owns: ['internal/tui/'] }, null),
  ];

  it('counts who came back, names each child from its path, and shows what it owns', () => {
    const d = delegation(kids);
    expect([d.count, d.finished, d.running, d.disjoint]).toEqual([4, 2, 2, true]);
    expect(d.children.map((c) => [c.name, c.owns, c.status, c.meta])).toEqual([
      ['alpha', 'internal/alpha/', 'ok', 'leu 9'],
      ['bravo', 'internal/bravo/', 'failed', 'não respondeu'],
      ['store', 'internal/store/', 'running', '…'],
      ['tui', 'internal/tui/', 'running', '…'],
    ]);
    expect(d.children[1]?.reason).toBe('the delegated turn failed: context deadline exceeded');
  });

  it('leaves a running child without a bar, because nothing measures how far it got', () => {
    expect(delegation(kids).children.map((c) => c.bar)).toEqual([1, 1, null, null]);
  });

  it('is disjoint only when every child owns paths no other child owns', () => {
    const shared = [
      call('explore', { path: 'a', owns: ['internal/store/'] }, null),
      call('explore', { path: 'b', owns: ['internal/store/sqlite.go'] }, null),
    ];
    expect(delegation(shared).disjoint).toBe(false);
    expect(delegation([call('explore', { path: 'a' }, null), call('explore', { path: 'b' }, null)]).disjoint).toBe(false);
  });
});

describe('the activity line', () => {
  it('names the phase of what runs, and never a verb with nothing beside it', () => {
    const t = (entries: ToolCall[]) => activity(entries.map((c, i) => ({ kind: 'tool' as const, seq: i + 1, at: '', call: c })));
    expect(t([])).toEqual({ phase: null, fact: { kind: 'none' } });
    expect(t([call('bash', { command: 'go test ./...' }, null)])).toEqual({
      phase: 'running',
      fact: { kind: 'tool', name: 'bash', target: 'go test ./...' },
    });
    expect(
      t([call('explore', { path: 'internal/store/' }, null), call('explore', { path: 'internal/tui/' }, null)]),
    ).toEqual({ phase: 'delegating', fact: { kind: 'children', names: ['store', 'tui'] } });
  });
});

describe('the seal', () => {
  it('maps the verification on the wire, and says nothing for a clean turn', () => {
    expect(sealOf({ verification: 'passed', met: ['go test', 'go vet'] })).toEqual({ text: '✓ verificado · 2 critérios', tone: 'ok' });
    expect(sealOf({ verification: 'failed', unmet: ['go test'] })?.tone).toBe('err');
    expect(sealOf({ verification: 'stale' })?.text).toMatch(/não conferido/);
    expect(sealOf({ verification: 'unavailable' })?.text).toMatch(/não conferido/);
    expect(sealOf({ verification: 'clean' })).toBeNull();
  });

  it('says when the turn wrote where the work is measured, and a pass no longer reads as a plain one', () => {
    expect(sealOf({ verification: 'passed', met: ['go test'], touched_protected: ['internal/x_test.go'] })).toEqual({
      text: '✓ verificado · 1 critério · tocou a régua: internal/x_test.go',
      tone: 'warn',
    });
    expect(sealOf({ verification: 'failed', unmet: ['go test'], touched_protected: ['a', 'b'] })).toEqual({
      text: '✗ não verificado · go test · tocou a régua: a, b',
      tone: 'err',
    });
    // Never left out when present, even where the seal itself says nothing.
    expect(sealOf({ verification: 'clean', touched_protected: ['done.toml'] })).toEqual({ text: '⚠ tocou a régua: done.toml', tone: 'warn' });
  });
});
