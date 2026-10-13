import { describe, expect, it } from 'vitest';
import { filesTouched } from '../../src/state/context';
import { done, log } from './helpers';

describe('the files a conversation changed', () => {
  it('lists each file written or edited once, newest first, with the lines the tools reported', () => {
    const v = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'fila.md' } })
      .add('tool.completed', done('c1', { added: 12 }))
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c2', name: 'read', input: { path: 'README.md' } })
      .add('tool.completed', done('c2'))
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c3', name: 'edit', input: { path: 'parser.ts' } })
      .add('tool.completed', done('c3', { added: 3, removed: 1 }))
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c4', name: 'edit', input: { path: 'fila.md' } })
      .add('tool.completed', done('c4', { added: 2, removed: 2 }))
      .fold();
    expect(filesTouched(v.entries)).toEqual([
      { path: 'fila.md', added: 14, removed: 2, failed: false },
      { path: 'parser.ts', added: 3, removed: 1, failed: false },
    ]);
  });

  it('says a write that failed, and leaves out calls with no path', () => {
    const v = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'x.go' } })
      .add('tool.completed', { ...done('c1'), ok: false, output: 'permission denied' })
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c2', name: 'write', input: {} })
      .fold();
    expect(filesTouched(v.entries)).toEqual([{ path: 'x.go', added: 0, removed: 0, failed: true }]);
  });
});
