// The slash commands the user and the project wrote (`GET /v1/commands`, D50),
// read at the boundary the way the skills are: an answer that does not match
// the generated type is a fact to say, never a list to guess from.

import type * as P from './generated';
import { check, isRecord, req, type Shape } from './validate';

const COMMAND: Shape<P.CommandInfo> = {
  name: req('string'),
  description: req('string'),
  source: req('string'),
  path: req('string'),
  body: req('string'),
};

type Decoded<T> = { ok: true; value: T } | { ok: false; reason: string };

/** Reads the answer of `GET /v1/commands`, as it came off the wire. A null list is the same empty list. */
export function decodeCommands(raw: unknown): Decoded<P.CommandsResponse> {
  const fail = (reason: string): Decoded<P.CommandsResponse> => ({ ok: false, reason: `lista de comandos: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  const commands = raw.commands ?? [];
  if (!Array.isArray(commands)) return fail('commands deveria ser uma lista');
  for (let i = 0; i < commands.length; i++) {
    const why = check(COMMAND, commands[i], `commands[${i}]`);
    if (why) return fail(why);
  }
  const overrides = raw.overrides ?? [];
  if (!Array.isArray(overrides) || !overrides.every((o) => typeof o === 'string')) return fail('overrides deveria ser uma lista de nomes');
  return { ok: true, value: { commands: commands as P.CommandInfo[], overrides: overrides as string[] } };
}
