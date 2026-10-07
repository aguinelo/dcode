// What a session can ask for (`GET /v1/models`), read at the boundary the way
// the list and every event are: an answer that does not match the generated
// type is a fact to say, never a menu to guess from.

import type * as P from './generated';
import { check, isRecord, opt, req, type Shape } from './validate';

const CHOICE: Shape<P.ModelChoice> = {
  name: req('string'),
  model: req('string'),
  family: req('string'),
  transport: req('string'),
  base_url: opt('string'),
  window: opt('number'),
  measured: req('boolean'),
  notice: opt('string'),
};

export type DecodedModels = { ok: true; models: P.ModelsResponse } | { ok: false; reason: string };

/** Reads the answer of `GET /v1/models`, as it came off the wire. */
export function decodeModels(raw: unknown): DecodedModels {
  const fail = (reason: string): DecodedModels => ({ ok: false, reason: `lista de modelos: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  const why = check(CHOICE, raw.default, 'default');
  if (why) return fail(why);
  // The daemon sends [] for none; a null would be the same empty list.
  const profiles = raw.profiles ?? [];
  if (!Array.isArray(profiles)) return fail('profiles deveria ser uma lista');
  for (let i = 0; i < profiles.length; i++) {
    const w = check(CHOICE, profiles[i], `profiles[${i}]`);
    if (w) return fail(w);
  }
  return { ok: true, models: { default: raw.default as P.ModelChoice, profiles: profiles as P.ModelChoice[] } };
}
