import { GitPullRequest } from 'lucide-react';
import { useState } from 'react';
import type * as P from '../protocol/generated';
import { agentTree, childStateView, ciView, liveCount, mainHealthView, ownsSummary, questionsOf, usageLine, type AgentList } from '../state/agents';
import { cardFor } from '../state/card';
import { NOTE_LIMIT, takesNote } from '../state/questions';
import type { Outcome } from './window';

const KIND: Record<string, string> = { merge: 'merge', design: 'desenho', ceiling: 'teto' };

/** Sends the person's answer, with the note or "", and says whether the daemon took it. */
export type AnswerQuestion = (questionId: string, answer: string, note: string) => Promise<Outcome<null>>;

/**
 * One question of the queue (D66): what it asks, the recommended option and
 * why, the sensitive paths of a merge, each option the daemon sent as a
 * button, and a note where the answer takes one. Nothing is optimistic: the
 * question leaves when the daemon took the answer, and a refusal is said here.
 */
function QuestionItem({ q, of, onAnswer }: { q: P.Question; of: string | null; onAnswer?: AnswerQuestion }) {
  const [note, setNote] = useState('');
  const [sending, setSending] = useState<string | null>(null);
  const [why, setWhy] = useState<string | null>(null);
  const recommended = q.options.find((o) => o.id === q.recommended)?.label ?? q.recommended;
  const send = (answer: string) => {
    if (!onAnswer || sending) return;
    setSending(answer);
    setWhy(null);
    void onAnswer(q.id, answer, note).then((out) => {
      setSending(null);
      if (!out.ok) setWhy(out.why);
    });
  };
  return (
    <li data-question-id={q.id}>
      <span className="agents-q-head">
        <span className="tone-warn">◆ {KIND[q.kind] ?? q.kind}</span>
        {of && <span className="tone-faint"> · {of}</span>}
      </span>
      <span className="agents-q-text">{q.text}</span>
      <span className="agents-q-why tone-dim">
        recomenda <b>{recommended}</b> — {q.why}
      </span>
      {q.sensitive && q.sensitive.length > 0 && <span className="agents-q-why tone-warn mono">sensível: {q.sensitive.join(', ')}</span>}
      {takesNote(q) && (
        <textarea
          className="agents-q-note-field"
          rows={2}
          value={note}
          maxLength={NOTE_LIMIT}
          placeholder="Nota opcional: o porquê, o que falta"
          aria-label={`Nota da resposta a ${q.text}`}
          disabled={!onAnswer || sending !== null}
          onChange={(e) => setNote(e.target.value)}
        />
      )}
      <span className="agents-q-options" role="group" aria-label="Respostas">
        {q.options.map((o) => (
          <button
            key={o.id}
            type="button"
            className={o.id === q.recommended ? 'agents-q-option recommended' : 'agents-q-option'}
            data-answer={o.id}
            data-recommended={o.id === q.recommended ? 'true' : undefined}
            disabled={!onAnswer || sending !== null}
            title={o.id === q.recommended ? 'A recomendada' : undefined}
            onClick={() => send(o.id)}
          >
            {sending === o.id ? 'enviando…' : o.label}
          </button>
        ))}
      </span>
      {why && <span className="agents-q-why tone-err" role="alert">{why}</span>}
    </li>
  );
}

/**
 * Takes the person to a card in the conversation (D67): scrolled into view and
 * focused, so ↑, ↓ and ↵ answer it. False when the card is not drawn.
 */
function focusCard(id: string): boolean {
  const el = document.querySelector<HTMLElement>(`[data-card-id="${CSS.escape(id)}"]`);
  if (!el) return false;
  el.scrollIntoView({ block: 'center', behavior: 'smooth' });
  el.focus({ preventScroll: true });
  return true;
}

/** The queue of what waits for the person, opened from "◆ N perguntas" when no card holds it, answered here (D66). */
export function QuestionQueue({ questions, titleOf, onAnswer }: { questions: P.Question[]; titleOf: (id: string) => string | null; onAnswer?: AnswerQuestion }) {
  return (
    <ul className="agents-questions" aria-label="Perguntas esperando você">
      {questions.map((q) => (
        <QuestionItem key={q.id} q={q} of={q.child_id ? titleOf(q.child_id) : null} onAnswer={onAnswer} />
      ))}
      <li className="agents-q-note tone-faint">
        {onAnswer
          ? 'A primeira resposta vale — desta janela, de outra ou da TUI; a pergunta sai quando o daemon a toma.'
          : 'Responder pede o daemon, e esta janela não fala com nenhum.'}
      </li>
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
  onAnswer,
  readError,
  cards = [],
}: {
  agents: AgentList;
  /** The question cards of the conversation (D67): "◆ N perguntas" takes the person to the one that holds them. */
  cards?: readonly P.QuestionCard[];
  /** The orchestrator's session id: its children are the tree's first level. */
  rootId: string;
  /** Whether a child's conversation is in the window's list, to be opened. */
  canOpen: (id: string) => boolean;
  onOpen: (id: string) => void;
  /** Answers a question of the queue (D66); absent, the queue only shows. */
  onAnswer?: AnswerQuestion;
  /** Why the daemon's /children or /questions could not be read; the events' list is what shows then. */
  readError?: string | null;
}) {
  const [queue, setQueue] = useState(false);
  const nodes = agentTree(agents, rootId);
  const n = agents.questions.length;
  const live = liveCount(agents);
  const main = agents.main ? mainHealthView(agents.main) : null;
  const titleOf = (id: string) => agents.children.find((c) => c.id === id)?.title ?? null;
  // A card holds the questions: they are answered there, in the conversation, and the queue is for none (D67).
  const target = cardFor(cards, agents.questions);
  return (
    <>
      <div className="agents-head">
        {/* Project.autonomy is "step", always, in this slice (.p §2). */}
        <span className="agents-mode" title="Todo merge pergunta antes.">
          passo a passo
        </span>
        {main && <span className={`agents-main-health tone-${main.tone}`}>{main.word}</span>}
        {n > 0 && (
          <button
            type="button"
            className="agents-asks"
            aria-expanded={target ? undefined : queue}
            data-card-target={target ?? undefined}
            title={target ? 'Ir ao cartão de perguntas na conversa' : undefined}
            onClick={() => {
              if (!target || !focusCard(target)) setQueue((q) => !q);
            }}
          >
            ◆ {n} {n === 1 ? 'pergunta' : 'perguntas'}
          </button>
        )}
      </div>
      {readError && (
        <p className="crew-quiet tone-err" role="alert">
          {readError} — a lista abaixo é a dos eventos.
        </p>
      )}
      {queue && n > 0 && !target && <QuestionQueue questions={agents.questions} titleOf={titleOf} onAnswer={onAnswer} />}
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
