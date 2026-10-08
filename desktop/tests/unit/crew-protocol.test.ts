import { describe, expect, it } from 'vitest';
import { decodeMemory, decodeSkills } from '../../src/protocol/crew';
import { memoryGroups } from '../../src/state/crew';

const skills = {
  enabled: true,
  skills: [
    { name: 'release', when_to_use: 'ao publicar', triggers: ['release'], source: 'project', path: 'release/SKILL.md', held: false },
    { name: 'yolo', when_to_use: 'nunca', source: 'user', path: 'yolo.md', held: true, claims: ['claims the sandbox is off (sandbox is disabled)'] },
  ],
  notices: [{ source: 'project', path: 'draft.md', reason: 'has no `when_to_use` line' }],
};

const memory = {
  path: '.dcode/memory.md',
  exists: true,
  enabled: true,
  max_entries: 40,
  entries: [
    { kind: 'convention', subject: 'commits em inglês', stale: false, shown: true },
    { kind: 'gotcha', subject: 'o socket', body: 'fica em /tmp', learned: '2026-08-18', commit: 'abc1234', stale: true, shown: true },
    { kind: 'decision', subject: 'daemon único', stale: false, shown: false },
    { kind: 'gotcha', subject: 'a CSP', stale: false, shown: true },
  ],
  malformed: [{ line: '## diary: hoje', reason: 'unknown kind "diary"' }],
};

describe('the answer of GET /v1/skills', () => {
  it('reads it as it came, triggers and claims optional', () => {
    expect(decodeSkills(skills)).toEqual({ ok: true, value: skills });
  });

  it('takes a null list for an empty one', () => {
    expect(decodeSkills({ enabled: false, skills: null, notices: null })).toEqual({ ok: true, value: { enabled: false, skills: [], notices: [] } });
  });

  it('says what does not match, never guesses', () => {
    expect(decodeSkills([])).toEqual({ ok: false, reason: 'lista de skills: a resposta não é um objeto' });
    const bad = decodeSkills({ ...skills, skills: [{ name: 'x', source: 'user', path: 'x.md', held: false }] });
    expect(bad.ok).toBe(false);
    expect(!bad.ok && bad.reason).toContain('skills[0]');
  });
});

describe('the answer of GET /v1/memory', () => {
  it('reads it as it came, unreadable only when there', () => {
    expect(decodeMemory(memory)).toEqual({ ok: true, value: memory });
    const broken = decodeMemory({ ...memory, entries: [], unreadable: 'permission denied' });
    expect(broken.ok && broken.value.unreadable).toBe('permission denied');
  });

  it('says what does not match', () => {
    expect(decodeMemory({ ...memory, max_entries: '40' })).toEqual({ ok: false, reason: 'memória: max_entries deveria ser número' });
    const bad = decodeMemory({ ...memory, malformed: [{ line: 'x' }] });
    expect(!bad.ok && bad.reason).toContain('malformed[0]');
  });
});

describe('the memories by kind', () => {
  it('groups gotchas, decisions and conventions in that order, each in file order, and keeps an unknown kind', () => {
    const groups = memoryGroups([...memory.entries, { kind: 'diary', subject: 'hoje', stale: false, shown: true }]);
    expect(groups.map((g) => [g.label, g.entries.map((e) => e.subject)])).toEqual([
      ['Armadilhas', ['o socket', 'a CSP']],
      ['Decisões', ['daemon único']],
      ['Convenções', ['commits em inglês']],
      ['diary', ['hoje']],
    ]);
  });
});
