import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { decodeChildren, decodeProject, decodeQuestion, decodeQuestions } from '../../src/protocol/projects';
import { QuestionQueue } from '../../src/renderer/AgentsPanel';
import { answerRefusal, openRefusal, takesNote } from '../../src/state/questions';

// The queue answered from the Agentes tab (D66): each question with its
// options — the recommended one marked —, a note where the answer takes one,
// and what the daemon said when it did not take the answer.

const merge: P.Question = {
  id: 'q-merge',
  kind: 'merge',
  text: 'Merge #521?',
  recommended: 'merge',
  why: 'revisão aprovou',
  child_id: 'c-a',
  options: [
    { id: 'merge', label: 'Merge' },
    { id: 'return', label: 'Devolver' },
    { id: 'close', label: 'Fechar o filho' },
  ],
  sensitive: ['internal/session/reopen.go'],
  asked: '2026-10-18T10:00:00Z',
};
const ceiling: P.Question = {
  id: 'q-ceiling',
  kind: 'ceiling',
  text: 'Passou do teto de tokens.',
  recommended: 'stop',
  why: 'nenhum filho novo até a resposta',
  options: [
    { id: 'continue', label: 'Continuar' },
    { id: 'stop', label: 'Parar' },
  ],
  asked: '2026-10-18T10:01:00Z',
};

const render = (answer?: (q: string, a: string, n: string) => Promise<{ ok: true; value: null } | { ok: false; why: string }>) =>
  renderToStaticMarkup(<QuestionQueue questions={[merge, ceiling]} titleOf={() => 'Sessão com pai'} onAnswer={answer} />);

describe('the question queue', () => {
  it('offers every option the daemon sent, the recommended one marked', () => {
    const html = render(async () => ({ ok: true, value: null }));
    for (const id of ['merge', 'return', 'close', 'continue', 'stop']) expect(html).toContain(`data-answer="${id}"`);
    expect(html.match(/data-recommended="true"/g)).toHaveLength(2);
    expect(html).toContain('sensível: internal/session/reopen.go');
    expect(html).not.toContain('Responder daqui chega');
  });

  it('takes a note on a merge and not on the ceiling', () => {
    expect(takesNote(merge)).toBe(true);
    expect(takesNote(ceiling)).toBe(false);
    const html = render(async () => ({ ok: true, value: null }));
    expect(html.match(/<textarea/g)).toHaveLength(1);
  });

  it('says it cannot answer when nothing would send the answer', () => {
    const html = render(undefined);
    expect(html).toContain('disabled');
    expect(html).toContain('Responder pede o daemon');
  });
});

describe('what the daemon said when it did not take it', () => {
  it('says another answer came first', () => {
    expect(answerRefusal('question_already_answered', 'question q1 was already answered "merge"')).toMatch(/já foi respondida.*q1 was already answered/);
    expect(answerRefusal('question_not_found', 'no q9')).toContain('não tem mais esta pergunta');
    expect(answerRefusal('internal', 'boom')).toBe('A resposta não foi enviada: boom');
  });

  it('names the keys to set when the daemon conducts no projects', () => {
    const why = openRefusal('orchestrator_unavailable', 'record.enabled is false: …', '/w/dcode');
    expect(why).toContain('/w/dcode');
    expect(why).toContain('record.enabled is false');
    expect(why).toContain('orchestrator.enabled = true');
    expect(why).toContain('record.enabled = true');
    expect(openRefusal('project_not_found', 'not a git tree', '/w/x')).toBe('A orquestradora de /w/x não abriu: not a git tree');
  });
});

describe('the answers of the project routes, read at the boundary', () => {
  const project = { id: 'p1', root: '/w', main: 'main', forge: 'local', autonomy: 'step', orchestrator_id: 's-orq', created: '2026-10-18T10:00:00Z' };

  it('reads a project, the children, the questions and an answered question', () => {
    expect(decodeProject(project)).toEqual({ ok: true, value: project });
    expect(decodeChildren({ children: [] })).toEqual({ ok: true, value: [] });
    // Go writes a nil slice as null: no children.
    expect(decodeChildren({ children: null })).toEqual({ ok: true, value: [] });
    expect(decodeQuestions({ questions: [merge] })).toEqual({ ok: true, value: { questions: [merge], cards: [] } });
    expect(decodeQuestion({ ...merge, answer: 'merge', answered: '2026-10-18T10:05:00Z' }).ok).toBe(true);
  });

  it('says a shape it does not have, never guesses from it', () => {
    expect(decodeProject({ id: 'p1' }).ok).toBe(false);
    expect(decodeChildren({ children: [{ id: 'c' }] }).ok).toBe(false);
    expect(decodeQuestions({ questions: 'x' }).ok).toBe(false);
    expect(decodeQuestion(null).ok).toBe(false);
  });
});
