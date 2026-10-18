// The answers of the project routes (orchestration .p §5.1), read at the
// boundary like the events: a project opened for a folder, the session that
// conducts it, its children and the questions that wait for the person. An
// answer that does not match the generated type is a fact to say, never a
// value to guess from (D66).

import { SESSION } from './events';
import type * as P from './generated';
import { CHILD, QUESTION, QUESTION_CARD } from './orchestrate';
import { check, isRecord, opt, req, type Shape } from './validate';

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

const ANSWER_RESULT: Shape<P.AnswerResult> = {
  question_id: req('string'),
  status: req('string'),
  code: opt('string'),
  message: opt('string'),
  question: opt('object', QUESTION),
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
const questionList = list(QUESTION, 'questions');
/** `POST /v1/sessions/{id}/questions/{qid}`: the question as the answer left it. */
export const decodeQuestion = one(QUESTION, 'pergunta');
/** `GET /v1/sessions/{id}/questions`: the cards the queue is shown in, open and closed (.p §5.5). */
export const decodeCards = list(QUESTION_CARD, 'cards');

/** `POST /v1/sessions/{id}/questions/answers`: a result per answer and per skip, and the cards they changed (D67). */
export function decodeAnswers(raw: unknown): Decoded<P.AnswerQuestionsResponse> {
  const results = list(ANSWER_RESULT, 'results')(raw);
  if (!results.ok) return results;
  const cards = decodeCards(raw);
  return cards.ok ? { ok: true, value: { results: results.value, cards: cards.value } } : cards;
}

/** `GET /v1/sessions/{id}/questions`: the queue, answered or not, and the cards it is shown in. */
export function decodeQuestions(raw: unknown): Decoded<{ questions: P.Question[]; cards: P.QuestionCard[] }> {
  const questions = questionList(raw);
  if (!questions.ok) return questions;
  const cards = decodeCards(raw);
  return cards.ok ? { ok: true, value: { questions: questions.value, cards: cards.value } } : cards;
}
