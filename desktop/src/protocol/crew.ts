// What a session in a workspace has as skills (`GET /v1/skills`) and as memory
// (`GET /v1/memory`), read at the boundary the way the models are: an answer
// that does not match the generated type is a fact to say, never a list to
// guess from.

import type * as P from './generated';
import { check, isRecord, opt, req, type Shape } from './validate';

const SKILL: Shape<P.SkillInfo> = {
  name: req('string'),
  when_to_use: req('string'),
  triggers: opt('array', 'string'),
  source: req('string'),
  path: req('string'),
  held: req('boolean'),
  claims: opt('array', 'string'),
};

const NOTICE: Shape<P.SkillNotice> = { source: req('string'), path: req('string'), reason: req('string') };

const ENTRY: Shape<P.MemoryEntry> = {
  kind: req('string'),
  subject: req('string'),
  body: opt('string'),
  learned: opt('string'),
  commit: opt('string'),
  stale: req('boolean'),
  shown: req('boolean'),
};

const MALFORMED: Shape<P.MemoryMalformed> = { line: req('string'), reason: req('string') };

type Decoded<T> = { ok: true; value: T } | { ok: false; reason: string };

/** Each item of a list against its shape; the daemon sends [] for none, and a null is the same empty list. */
function items<T>(raw: unknown, shape: Shape<T>, name: string): Decoded<T[]> {
  const list = raw ?? [];
  if (!Array.isArray(list)) return { ok: false, reason: `${name} deveria ser uma lista` };
  for (let i = 0; i < list.length; i++) {
    const why = check(shape, list[i], `${name}[${i}]`);
    if (why) return { ok: false, reason: why };
  }
  return { ok: true, value: list as T[] };
}

/** Reads the answer of `GET /v1/skills`, as it came off the wire. */
export function decodeSkills(raw: unknown): Decoded<P.SkillsResponse> {
  const fail = (reason: string): Decoded<P.SkillsResponse> => ({ ok: false, reason: `lista de skills: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  if (typeof raw.enabled !== 'boolean') return fail('enabled deveria ser booleano');
  const skills = items(raw.skills, SKILL, 'skills');
  if (!skills.ok) return fail(skills.reason);
  const notices = items(raw.notices, NOTICE, 'notices');
  if (!notices.ok) return fail(notices.reason);
  return { ok: true, value: { enabled: raw.enabled, skills: skills.value, notices: notices.value } };
}

/** Reads the answer of `GET /v1/memory`, as it came off the wire. */
export function decodeMemory(raw: unknown): Decoded<P.MemoryResponse> {
  const fail = (reason: string): Decoded<P.MemoryResponse> => ({ ok: false, reason: `memória: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  if (typeof raw.path !== 'string') return fail('path deveria ser texto');
  if (typeof raw.exists !== 'boolean') return fail('exists deveria ser booleano');
  if (typeof raw.enabled !== 'boolean') return fail('enabled deveria ser booleano');
  if (typeof raw.max_entries !== 'number') return fail('max_entries deveria ser número');
  if (raw.unreadable !== undefined && typeof raw.unreadable !== 'string') return fail('unreadable deveria ser texto');
  const entries = items(raw.entries, ENTRY, 'entries');
  if (!entries.ok) return fail(entries.reason);
  const malformed = items(raw.malformed, MALFORMED, 'malformed');
  if (!malformed.ok) return fail(malformed.reason);
  return {
    ok: true,
    value: {
      path: raw.path,
      exists: raw.exists,
      enabled: raw.enabled,
      max_entries: raw.max_entries,
      entries: entries.value,
      malformed: malformed.value,
      ...(raw.unreadable ? { unreadable: raw.unreadable } : {}),
    },
  };
}
