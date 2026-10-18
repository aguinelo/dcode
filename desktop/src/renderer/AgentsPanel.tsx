import { GitPullRequest } from 'lucide-react';
import { useState } from 'react';
import type * as P from '../protocol/generated';
import { agentTree, childStateView, ciView, liveCount, mainHealthView, ownsSummary, questionsOf, usageLine, type AgentList } from '../state/agents';

const KIND: Record<string, string> = { merge: 'merge', design: 'desenho', ceiling: 'teto' };

/** The queue of what waits for the person, opened from "◆ N perguntas". */
function Questions({ questions, titleOf }: { questions: P.Question[]; titleOf: (id: string) => string | null }) {
  return (
    <ul className="agents-questions" aria-label="Perguntas esperando você">
      {questions.map((q) => {
        const recommended = q.options.find((o) => o.id === q.recommended)?.label ?? q.recommended;
        const of = q.child_id ? titleOf(q.child_id) : null;
        return (
          <li key={q.id} data-question-id={q.id}>
            <span className="agents-q-head">
              <span className="tone-warn">◆ {KIND[q.kind] ?? q.kind}</span>
              {of && <span className="tone-faint"> · {of}</span>}
            </span>
            <span className="agents-q-text">{q.text}</span>
            <span className="agents-q-why tone-dim">
              recomenda <b>{recommended}</b> — {q.why}
            </span>
            {q.sensitive && q.sensitive.length > 0 && <span className="agents-q-why tone-warn mono">sensível: {q.sensitive.join(', ')}</span>}
          </li>
        );
      })}
      <li className="agents-q-note tone-faint">Responder daqui chega quando o app ler a fila do daemon; até lá, a resposta vai na conversa.</li>
    </ul>
  );
}

function Row({ child, depth, asks, refused, canOpen, onOpen }: {
  child: P.Child;
  depth: number;
  asks: P.Question[];
  refused: P.MergeRefused | undefined;
  canOpen: boolean;
  onOpen: (id: string) => void;
}) {
  const st = childStateView(child);
  const ci = child.pr ? ciView(child.pr.ci ?? '') : null;
  return (
    <li style={depth > 0 ? { paddingLeft: depth * 14 } : undefined}>
      <button
        type="button"
        className="agents-row"
        data-child-id={child.id}
        data-child-state={child.state}
        disabled={!canOpen}
        title={canOpen ? `Abrir a conversa de ${child.title}` : undefined}
        onClick={() => onOpen(child.id)}
      >
        <span className={`agents-dot tone-${st.tone}`} aria-hidden />
        <span className="agents-main">
          <span className="agents-line">
            <span className="agents-title">{child.title}</span>
            <span className={`agents-state tone-${st.tone}`}>{st.word}</span>
          </span>
          <span className="agents-line agents-sub">
            <span className="mono agents-owns" title={child.owns.join('\n')}>
              {ownsSummary(child.owns)}
            </span>
            {child.pr && (
              <span className="agents-pr">
                <GitPullRequest size={11} aria-hidden />
                {child.pr.number ? `#${child.pr.number}` : 'local'}
                {ci && (
                  <>
                    <span className={`agents-ci-dot tone-${ci.tone}`} aria-hidden />
                    <span className={`tone-${ci.tone}`}>{ci.word}</span>
                  </>
                )}
              </span>
            )}
          </span>
          <span className="agents-sub agents-usage mono">{usageLine(child.usage)}</span>
          {st.detail && asks.length === 0 && <span className={`agents-sub ${st.tone === 'warn' || st.tone === 'err' ? `tone-${st.tone}` : ''}`}>{st.detail}</span>}
          {asks.map((q) => (
            <span key={q.id} className="agents-ask tone-warn">
              ◆ {q.text}
            </span>
          ))}
          {refused && (
            <span className="agents-ask tone-err">
              merge recusado: {refused.code} — {refused.reason}
            </span>
          )}
          {!canOpen && <span className="agents-sub">sem conversa na lista desta janela</span>}
        </span>
      </button>
    </li>
  );
}

/**
 * The live agent list (orchestration RN-9) in the Crew's right pane of an
 * orchestrator's conversation, with the grammar of Mudanças (D41): one row
 * per child — its mark, title, state and tokens, its PR and CI by D60 and
 * D61 — and, on top, the mode and "◆ N perguntas", which opens the queue.
 * Only the wire (D8): child.*, question.* and main.health.
 */
export function AgentsPanel({
  agents,
  rootId,
  canOpen,
  onOpen,
}: {
  agents: AgentList;
  /** The orchestrator's session id: its children are the tree's first level. */
  rootId: string;
  /** Whether a child's conversation is in the window's list, to be opened. */
  canOpen: (id: string) => boolean;
  onOpen: (id: string) => void;
}) {
  const [queue, setQueue] = useState(false);
  const nodes = agentTree(agents, rootId);
  const n = agents.questions.length;
  const live = liveCount(agents);
  const main = agents.main ? mainHealthView(agents.main) : null;
  const titleOf = (id: string) => agents.children.find((c) => c.id === id)?.title ?? null;
  return (
    <>
      <div className="agents-head">
        {/* Project.autonomy is "step", always, in this slice (.p §2). */}
        <span className="agents-mode" title="Todo merge pergunta antes.">
          passo a passo
        </span>
        {main && <span className={`agents-main-health tone-${main.tone}`}>{main.word}</span>}
        {n > 0 && (
          <button type="button" className="agents-asks" aria-expanded={queue} onClick={() => setQueue((q) => !q)}>
            ◆ {n} {n === 1 ? 'pergunta' : 'perguntas'}
          </button>
        )}
      </div>
      {queue && n > 0 && <Questions questions={agents.questions} titleOf={titleOf} />}
      {nodes.length === 0 ? (
        <p className="crew-quiet">Esta orquestradora ainda não despachou nenhum agente.</p>
      ) : (
        <>
          <div className="crew-tab-sum">
            <span>
              {live} {live === 1 ? 'agente em curso' : 'agentes em curso'}
            </span>
            <span className="tone-faint">{nodes.length} na lista</span>
          </div>
          <ul className="agents-list">
            {nodes.map(({ child, depth }) => (
              <Row
                key={child.id}
                child={child}
                depth={depth}
                asks={questionsOf(agents, child.id)}
                refused={agents.refused[child.id]}
                canOpen={canOpen(child.id)}
                onOpen={onOpen}
              />
            ))}
          </ul>
        </>
      )}
      <p className="crew-quiet crew-foot-note">
        Lido de child.*, question.* e main.health (D8). Um fechado sai quando o daemon deixa de trazê-lo; um que falhou fica até alguém agir.
      </p>
    </>
  );
}
