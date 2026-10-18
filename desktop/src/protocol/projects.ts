// The answers of the project routes (orchestration .p §5.1), read at the
// boundary like the events: a project opened for a folder, the session that
// conducts it, its children and the questions that wait for the person. An
// answer that does not match the generated type is a fact to say, never a
// value to guess from (D66).

import { SESSION } from './events';
import type * as P from './generated';
import { CHILD, QUESTION } from './orchestrate';
import { check, isRecord, req, type Shape } from './validate';

type Decoded<T> = { ok: true; value: T } | { ok: false; reason: string };

const PROJECT: Shape<P.Project> = {
  id: req('string'),
  root: req('string'),
  main: req('string'),
  forge: req('string'),
  autonomy: req('string'),
  orchestrator_id: req('string'),
  created: req('string'),
};

function one<T>(shape: Shape<T>, what: string): (raw: unknown) => Decoded<T> {
  return (raw) => {
    const why = check(shape, raw, what);
    return why ? { ok: false, reason: why } : { ok: true, value: raw as T };
  };
}

function list<T>(shape: Shape<T>, key: string): (raw: unknown) => Decoded<T[]> {
  return (raw) => {
    if (!isRecord(raw)) return { ok: false, reason: `${key}: a resposta não é um objeto` };
    // Go writes a nil slice as null: none.
    const items = raw[key] ?? [];
    if (!Array.isArray(items)) return { ok: false, reason: `${key} deveria ser uma lista` };
    for (let i = 0; i < items.length; i++) {
      const why = check(shape, items[i], `${key}[${i}]`);
      if (why) return { ok: false, reason: why };
    }
    return { ok: true, value: items as T[] };
  };
}

/** `POST /v1/projects`: the folder's project, opened or found. */
export const decodeProject = one(PROJECT, 'projeto');
/** `GET /v1/projects/{id}/orchestrator`: the session that conducts it. */
export const decodeOrchestrator = one(SESSION, 'orquestradora');
/** `GET /v1/sessions/{id}/children`: the live ones and the closed inside their retention. */
export const decodeChildren = list(CHILD, 'children');
/** `GET /v1/sessions/{id}/questions`: answered or not. */
export const decodeQuestions = list(QUESTION, 'questions');
/** `POST /v1/sessions/{id}/questions/{qid}`: the question as the answer left it. */
export const decodeQuestion = one(QUESTION, 'pergunta');
