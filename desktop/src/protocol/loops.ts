// The `/loop` runs the daemon conducts (N4), read at the boundary the way the
// list of conversations is: a frame that does not match the generated type is
// a fact to say, never a value to guess from.

import { COMPLETION } from './events';
import type * as P from './generated';
import { check, isRecord, opt, req, type Shape } from './validate';

const VERDICT: Shape<P.LoopVerdict> = {
  worked: req('number'),
  met: req('number'),
  total: req('number'),
  unmet: opt('array', 'string'),
  unchecked: opt('array', 'string'),
  unfinished: opt('array', 'string'),
};

const SPEC_RUN: Shape<P.LoopSpecRun> = {
  path: req('string'),
  phase: req('string'),
  session_id: opt('string'),
  criteria: opt('number'),
  unmet: opt('number'),
  unavailable: opt('number'),
  error: opt('string'),
  proposal: opt('string'),
  proposal_criteria: opt('number'),
  proposal_summary: opt('string'),
  reason: opt('string'),
  standing: opt('object', COMPLETION),
};

const RUN: Shape<P.LoopRun> = {
  id: req('string'),
  workspace: req('string'),
  args: opt('string'),
  goal: opt('string'),
  protect: opt('array', 'string'),
  state: req('string'),
  specs: req('array', SPEC_RUN),
  current: req('number'),
  session_id: opt('string'),
  verdict: opt('object', VERDICT),
  warnings: opt('array', 'string'),
  error: opt('string'),
  started: req('string'),
  finished: opt('string'),
};

const FOLDER: Shape<P.SpecFolder> = {
  path: req('string'),
  criteria: req('number'),
  unmet: req('number'),
  unavailable: opt('number'),
  pending: req('boolean'),
  measured: req('boolean'),
  error: opt('string'),
};

/** One frame of the stream of loops, read. */
export type LoopListChange = { kind: 'snapshot'; loops: P.LoopRun[] } | { kind: 'changed'; loop: P.LoopRun };

/** The kinds this desktop reads, held to the generated constants by a test. */
export const LOOP_CHANGE_KINDS = ['snapshot', 'changed'] as const;

type Decoded<T> = { ok: true; value: T } | { ok: false; reason: string };

/** A run as the daemon sends it; Go writes a nil list as null, which is the empty one. */
function asRun(raw: unknown, path: string): Decoded<P.LoopRun> {
  const why = check(RUN, raw, path);
  if (why) return { ok: false, reason: why };
  const r = raw as P.LoopRun;
  return { ok: true, value: { ...r, specs: r.specs ?? [] } };
}

/** Reads one frame of `GET /v1/loops/events`, as it came off the wire. */
export function decodeLoopChange(raw: unknown): { ok: true; change: LoopListChange } | { ok: false; reason: string } {
  const fail = (reason: string) => ({ ok: false as const, reason: `lista de loops: ${reason}` });
  if (!isRecord(raw)) return fail('a mudança não é um objeto');
  switch (raw.kind) {
    case 'snapshot': {
      const list = raw.loops ?? [];
      if (!Array.isArray(list)) return fail('loops deveria ser uma lista');
      const loops: P.LoopRun[] = [];
      for (let i = 0; i < list.length; i++) {
        const got = asRun(list[i], `loops[${i}]`);
        if (!got.ok) return fail(got.reason);
        loops.push(got.value);
      }
      return { ok: true, change: { kind: 'snapshot', loops } };
    }
    case 'changed': {
      const got = asRun(raw.loop, 'loop');
      return got.ok ? { ok: true, change: { kind: 'changed', loop: got.value } } : fail(got.reason);
    }
    default:
      return fail(`tipo de mudança que esta versão não conhece: ${String(raw.kind)}`);
  }
}

/** Reads the run the daemon answered a start, a stop or a continue with. */
export function decodeLoopRun(raw: unknown): Decoded<P.LoopRun> {
  const got = asRun(raw, 'loop');
  return got.ok ? got : { ok: false, reason: `loop: ${got.reason}` };
}

/** Reads `GET /v1/specs`: the folders a project has, and what each declares. */
export function decodeSpecs(raw: unknown): Decoded<P.SpecFolder[]> {
  const fail = (reason: string): Decoded<P.SpecFolder[]> => ({ ok: false, reason: `pastas de spec: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  const list = raw.specs ?? [];
  if (!Array.isArray(list)) return fail('specs deveria ser uma lista');
  for (let i = 0; i < list.length; i++) {
    const why = check(FOLDER, list[i], `specs[${i}]`);
    if (why) return fail(why);
  }
  return { ok: true, value: list as P.SpecFolder[] };
}
