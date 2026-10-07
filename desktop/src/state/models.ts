// The model menu: what a conversation can continue on (D28). The daemon says
// what exists and whether each family was measured; the window only marks what
// the conversation runs on now. Nothing here guesses a model the daemon did not
// list: a name that is not a profile would lose the endpoint a local one needs.

import type * as P from '../protocol/generated';

export interface ModelOption {
  /** What to ask for: a profile's name, or the default's model. */
  name: string;
  /** What it runs: the model, and where when it is not the transport's own endpoint. */
  detail: string;
  /** What a session in this project gets when it asks for none. */
  isDefault: boolean;
  /** dcode's contracts were measured against its family. */
  measured: boolean;
  /** The daemon's own words when it was not: the family's admission, or why no session opens with it. */
  notice: string | null;
  /** The conversation runs on it now. */
  current: boolean;
}

/** Where a conversation runs: its model, and its endpoint — null when not known, empty for the transport's own. */
export interface Running {
  model: string;
  baseUrl: string | null;
}

function hostOf(url: string): string {
  try {
    return new URL(url).host || url;
  } catch {
    return url;
  }
}

function runsOn(c: P.ModelChoice, running: Running | null): boolean {
  if (!running || c.model !== running.model) return false;
  return running.baseUrl === null || (c.base_url ?? '') === running.baseUrl;
}

function optionOf(c: P.ModelChoice, isDefault: boolean, running: Running | null): ModelOption {
  const where = c.base_url ? ` · ${hostOf(c.base_url)}` : '';
  return {
    name: c.name,
    detail: isDefault ? `padrão do projeto${where}` : `${c.model}${where}`,
    isDefault,
    measured: c.measured,
    notice: c.measured ? null : (c.notice ?? null),
    current: runsOn(c, running),
  };
}

/** What the daemon's list says of the family a conversation runs on. */
export interface Measure {
  measured: boolean;
  family: string;
}

/**
 * Whether the family a conversation runs on was measured, as the daemon's list
 * says; null when the list says nothing of it. Measurement belongs to the
 * family, so it is matched by the family the session announced, else by the
 * model the list of conversations carries.
 */
export function measureOf(models: P.ModelsResponse | undefined, running: { model: string; family: string | null } | null): Measure | null {
  if (!models || !running) return null;
  const all = [models.default, ...models.profiles];
  const same = running.family ? all.filter((c) => c.family === running.family) : all.filter((c) => c.model === running.model);
  const first = same[0];
  if (!first) return null;
  // A choice that cannot build a session reads as not measured; another of the same family may.
  return { measured: same.some((c) => c.measured), family: first.family };
}

/** The default first, then each profile in the daemon's order (by name). */
export function menuOf(models: P.ModelsResponse, running: Running | null): ModelOption[] {
  return [optionOf(models.default, true, running), ...models.profiles.map((p) => optionOf(p, false, running))];
}
