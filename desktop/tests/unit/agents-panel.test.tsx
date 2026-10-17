import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { orchestraRecording } from '../../src/fixtures/orchestra-recording';
import { referenceStates } from '../../src/fixtures/scenes';
import { AgentsPanel } from '../../src/renderer/AgentsPanel';
import { emptyAgents } from '../../src/state/agents';
import { applyRaw, emptySessions } from '../../src/state/sessions';

// The panel the orchestrator's conversation shows on the Crew's right
// (RN-9): one row per child, its state in words and tone, its tokens, its PR
// and CI, what it owns, and the questions waiting for the person.

const sessions = applyRaw(emptySessions, orchestraRecording.events);
const orq = sessions.byId['s-orq']!;

const render = (agents = orq.agents, canOpen: (id: string) => boolean = () => true) =>
  renderToStaticMarkup(<AgentsPanel agents={agents} rootId="s-orq" canOpen={canOpen} onOpen={() => {}} />);

describe('the agents panel', () => {
  it('replays an orchestrator with a child in each of the eight states', () => {
    expect(orq.info?.role).toBe('orchestrator');
    expect(new Set(orq.agents.children.map((c) => c.state))).toEqual(
      new Set(['dispatched', 'working', 'awaiting_person', 'in_review', 'pr', 'merged', 'closed', 'failed']),
    );
    expect(Object.keys(referenceStates)).toEqual(expect.arrayContaining(['crew-orquestra', 'crew-orquestra-vazia']));
  });

  it('draws one row per child with its state, tokens, PR and CI', () => {
    const html = render();
    expect(html.match(/data-child-id=/g)).toHaveLength(orq.agents.children.length);
    for (const word of ['despachado', 'trabalhando', 'esperando você', 'em revisão', 'merged', 'fechado', 'falhou']) expect(html).toContain(word);
    expect(html).toContain('entrada');
    expect(html).toContain('#518');
    expect(html).toContain('CI rodando');
    expect(html).not.toMatch(/\$|US\$|dólar/);
  });

  it('surfaces the questions waiting for the person', () => {
    const html = render();
    const n = orq.agents.questions.length;
    expect(n).toBeGreaterThan(0);
    expect(html).toContain(`◆ ${n} ${n === 1 ? 'pergunta' : 'perguntas'}`);
    expect(html).toContain('passo a passo');
  });

  it('says why a child cannot be opened when its conversation is not listed', () => {
    const html = render(orq.agents, () => false);
    expect(html).toContain('disabled');
    expect(html).toContain('A conversa deste agente não está na lista');
  });

  it('says an orchestrator with no agents yet', () => {
    const html = render(emptyAgents);
    expect(html).toContain('Esta orquestradora ainda não despachou nenhum agente.');
    expect(html).not.toContain('data-child-id');
  });
});
