import { afterEach, describe, expect, it } from 'vitest';
import { BATCH_LIMIT, NOTE_LIMIT, projectId, questionAnswer, questionAnswers, questionId } from '../../src/main/inputs';
import { Requests } from '../../src/main/requests';
import { fakeDaemon, tempSocket, type FakeOptions } from './fake-daemon';

// The window acting on a project's orchestrator (D66): opening it for a folder,
// reading its children and questions, and the person's answer — each checked
// here before it reaches the daemon, and sent on the route the daemon serves.

function refused(got: { ok: boolean; refusal?: { code: string; message: string } }): string {
  expect(got.ok).toBe(false);
  expect(got.refusal?.code).toBe('invalid_input');
  return got.refusal?.message ?? '';
}

describe('what an answer to a question carries', () => {
  it('is an option id and, when written, a note', () => {
    expect(questionAnswer('merge', undefined)).toEqual({ ok: true, value: { answer: 'merge' } });
    expect(questionAnswer('return', ' faltou o teste ')).toEqual({ ok: true, value: { answer: 'return', note: 'faltou o teste' } });
    // An empty note is no note: the daemon reads an absent one.
    expect(questionAnswer('close', '   ')).toEqual({ ok: true, value: { answer: 'close' } });
  });

  it('is refused when it is not one the daemon would take', () => {
    expect(refused(questionAnswer('', undefined))).toContain('resposta');
    expect(refused(questionAnswer(3, undefined))).toContain('resposta');
    expect(refused(questionAnswer('x'.repeat(65), undefined))).toContain('64');
    expect(refused(questionAnswer('merge', 7))).toContain('nota');
    expect(refused(questionAnswer('merge', 'é'.repeat(NOTE_LIMIT + 1)))).toContain(String(NOTE_LIMIT));
  });

  it('names a question and a project in the daemon’s alphabet, since each becomes a path there', () => {
    expect(questionId('q-1a2b')).toEqual({ ok: true, value: 'q-1a2b' });
    expect(refused(questionId('../x'))).toContain('pergunta');
    expect(projectId('0123456789abcdef01234567')).toEqual({ ok: true, value: '0123456789abcdef01234567' });
    expect(refused(projectId('a/b'))).toContain('projeto');
  });
});

describe('what the confirm of a card carries (D67)', () => {
  it('is each answer checked as one answer is, and the skipped with their card', () => {
    expect(questionAnswers({ card_id: 'c1', answers: [{ question_id: 'q1', answer: 'merge' }, { question_id: 'q2', answer: 'other', note: ' espera o CI ' }], skipped: ['q3'] })).toEqual({
      ok: true,
      value: { card_id: 'c1', answers: [{ question_id: 'q1', answer: 'merge' }, { question_id: 'q2', answer: 'other', note: 'espera o CI' }], skipped: ['q3'] },
    });
    expect(questionAnswers({ answers: [{ question_id: 'q1', answer: 'stop' }] })).toEqual({ ok: true, value: { answers: [{ question_id: 'q1', answer: 'stop' }] } });
  });

  it('is refused when the daemon would refuse the whole batch, or an answer would not be one', () => {
    expect(refused(questionAnswers('x'))).toContain('objeto');
    expect(refused(questionAnswers({ answers: [] }))).toContain('nenhuma resposta');
    expect(refused(questionAnswers({ answers: [], skipped: ['q1'] }))).toContain('id do cartão');
    expect(refused(questionAnswers({ card_id: '../c', answers: [{ question_id: 'q1', answer: 'merge' }] }))).toContain('cartão');
    expect(refused(questionAnswers({ answers: [{ question_id: '../q', answer: 'merge' }] }))).toContain('pergunta');
    expect(refused(questionAnswers({ answers: [{ question_id: 'q1', answer: '' }] }))).toContain('resposta');
    expect(refused(questionAnswers({ answers: [{ question_id: 'q1', answer: 'merge', note: 'é'.repeat(NOTE_LIMIT + 1) }] }))).toContain(String(NOTE_LIMIT));
    expect(refused(questionAnswers({ card_id: 'c1', answers: [], skipped: ['a/b'] }))).toContain('pergunta');
    const many = Array.from({ length: BATCH_LIMIT + 1 }, (_, i) => ({ question_id: `q${i}`, answer: 'merge' }));
    expect(refused(questionAnswers({ answers: many }))).toContain(String(BATCH_LIMIT));
  });
});

const cleanups: (() => void | Promise<void>)[] = [];
afterEach(async () => {
  for (const c of cleanups.splice(0).reverse()) await c();
});

async function daemon(projects: FakeOptions['projects']) {
  const t = tempSocket();
  cleanups.push(t.remove);
  const d = await fakeDaemon(t.socket, { projects, sessions: { 's-orq': 1 } });
  cleanups.push(() => d.kill());
  return { d, ask: new Requests(() => ({ state: 'connected', version: 'x', socket: d.socket, started: false })) };
}

const project = { id: 'p1', root: '/w/dcode', main: 'main', forge: 'local', autonomy: 'step', orchestrator_id: 's-orq', created: '2026-10-18T10:00:00Z' };

describe('the project routes', () => {
  it('opens a folder’s project and asks for its orchestrator', async () => {
    const { d, ask } = await daemon({ open: { status: 201, body: project }, orchestrator: { status: 200, body: { id: 's-orq' } } });
    expect(await ask.openProject('/w/dcode')).toEqual({ ok: true, value: project });
    expect(await ask.projectOrchestrator('p1')).toEqual({ ok: true, value: { id: 's-orq' } });
    expect(d.requests).toEqual(['POST /v1/projects', 'GET /v1/projects/p1/orchestrator']);
    expect(d.bodies).toEqual([{ workspace: '/w/dcode' }]);
  });

  it('passes the daemon’s refusal on, with its code', async () => {
    const off = { code: 'orchestrator_unavailable', message: 'orchestrator.enabled is false: the orchestrator is opt-in in this slice' };
    const { ask } = await daemon({ open: { status: 503, body: off } });
    expect(await ask.openProject('/w/dcode')).toEqual({ ok: false, refusal: off });
  });

  it('reads the children and the questions, and sends the answer with its note', async () => {
    const answered = { status: 200, body: { id: 'q1', answer: 'merge' } };
    const { d, ask } = await daemon({ open: { status: 201, body: project }, children: { status: 200, body: { children: [] } }, questions: { status: 200, body: { questions: [] } }, answer: answered });
    expect(await ask.listChildren('s-orq')).toEqual({ ok: true, value: { children: [] } });
    expect(await ask.listQuestions('s-orq')).toEqual({ ok: true, value: { questions: [] } });
    expect(await ask.answerQuestion('s-orq', 'q1', { answer: 'merge', note: 'ok' })).toEqual({ ok: true, value: answered.body });
    expect(await ask.answerQuestion('s-orq', 'q1', { answer: 'stop' })).toEqual({ ok: true, value: answered.body });
    expect(d.requests).toEqual(['GET /v1/sessions/s-orq/children', 'GET /v1/sessions/s-orq/questions', 'POST /v1/sessions/s-orq/questions/q1', 'POST /v1/sessions/s-orq/questions/q1']);
    expect(d.bodies).toEqual([{ answer: 'merge', note: 'ok' }, { answer: 'stop' }]);
  });

  it('confirms a card on its own route, the batch as it was checked', async () => {
    const answered = { status: 200, body: { results: [{ question_id: 'q1', status: 'answered' }], cards: [] } };
    const { d, ask } = await daemon({ open: { status: 201, body: project }, answers: answered });
    const batch = { card_id: 'c1', answers: [{ question_id: 'q1', answer: 'merge' }], skipped: ['q2'] };
    expect(await ask.answerQuestions('s-orq', batch)).toEqual({ ok: true, value: answered.body });
    expect(d.requests).toEqual(['POST /v1/sessions/s-orq/questions/answers']);
    expect(d.bodies).toEqual([batch]);
  });
});
