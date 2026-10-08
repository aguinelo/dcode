// What a session in a workspace has as skills (`GET /v1/skills`) and as memory
// (`GET /v1/memory`), read at the boundary the way the models are.
//
// The interfaces mirror internal/protocol's SkillsResponse and MemoryResponse
// (core PR #432, N6), written here by hand until that PR merges and tygo
// generates them into generated.ts; then these give way to the generated ones,
// and the shapes keep them honest as they do for every other type.

import { check, isRecord, opt, req, type Shape } from './validate';

export const SkillSourceUser = 'user';
export const SkillSourceProject = 'project';

export interface SkillInfo {
  name: string;
  when_to_use: string;
  triggers?: string[];
  /** `user` or `project`. */
  source: string;
  /** The skill's file, relative to its source's skills folder. */
  path: string;
  /** Reaches for the boundary: a session asks a person before loading it. */
  held: boolean;
  /** What it reaches for, in the words the question would use. */
  claims?: string[];
}

export interface SkillNotice {
  source: string;
  path: string;
  reason: string;
}

export interface SkillsResponse {
  /** Sessions in the workspace index skills at all (`behavior.skills_enabled`). */
  enabled: boolean;
  skills: SkillInfo[];
  notices: SkillNotice[];
}

export interface MemoryEntry {
  /** gotcha, decision or convention. */
  kind: string;
  subject: string;
  body?: string;
  learned?: string;
  commit?: string;
  /** The commit it was true at is no longer in the repository. */
  stale: boolean;
  /** A session reads it. */
  shown: boolean;
}

export interface MemoryMalformed {
  line: string;
  reason: string;
}

export interface MemoryResponse {
  /** The file, relative to the workspace. */
  path: string;
  exists: boolean;
  /** Sessions in the workspace read memory at all (`memory.enabled`). */
  enabled: boolean;
  /** How many memories reach a session (`memory.max_entries`). */
  max_entries: number;
  entries: MemoryEntry[];
  malformed: MemoryMalformed[];
  /** Why the file could not be read, when it could not. */
  unreadable?: string;
}

const SKILL: Shape<SkillInfo> = {
  name: req('string'),
  when_to_use: req('string'),
  triggers: opt('array', 'string'),
  source: req('string'),
  path: req('string'),
  held: req('boolean'),
  claims: opt('array', 'string'),
};

const NOTICE: Shape<SkillNotice> = { source: req('string'), path: req('string'), reason: req('string') };

const ENTRY: Shape<MemoryEntry> = {
  kind: req('string'),
  subject: req('string'),
  body: opt('string'),
  learned: opt('string'),
  commit: opt('string'),
  stale: req('boolean'),
  shown: req('boolean'),
};

const MALFORMED: Shape<MemoryMalformed> = { line: req('string'), reason: req('string') };

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
export function decodeSkills(raw: unknown): Decoded<SkillsResponse> {
  const fail = (reason: string): Decoded<SkillsResponse> => ({ ok: false, reason: `lista de skills: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  if (typeof raw.enabled !== 'boolean') return fail('enabled deveria ser booleano');
  const skills = items(raw.skills, SKILL, 'skills');
  if (!skills.ok) return fail(skills.reason);
  const notices = items(raw.notices, NOTICE, 'notices');
  if (!notices.ok) return fail(notices.reason);
  return { ok: true, value: { enabled: raw.enabled, skills: skills.value, notices: notices.value } };
}

/** Reads the answer of `GET /v1/memory`, as it came off the wire. */
export function decodeMemory(raw: unknown): Decoded<MemoryResponse> {
  const fail = (reason: string): Decoded<MemoryResponse> => ({ ok: false, reason: `memória: ${reason}` });
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
