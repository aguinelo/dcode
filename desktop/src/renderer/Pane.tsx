import type { Measure } from '../state/models';
import type { PaneView } from '../state/pane';
import { emptySession, type SessionView } from '../state/session';
import type { Row } from '../state/sidebar';
import { Field } from './Composer';
import { ConversationBody } from './ConversationBody';

/**
 * What the harness measured: each criterion, else the seal, else what is known
 * about done — and, when the daemon says so, that nobody measured the family
 * of the model doing the work.
 */
export function DoneRow({ view, measure }: { view: Pick<PaneView, 'lights' | 'seal' | 'doneNote' | 'touched'>; measure?: Measure | null }) {
  return (
    <div className="done-row">
      {view.lights.map((l) => (
        <span key={l.name} className={`crit ${l.state}`} title={l.state === 'unavailable' ? 'não pôde rodar' : undefined}>
          {l.state === 'met' ? '✓' : l.state === 'unmet' ? '✗' : '?'} {l.name}
        </span>
      ))}
      {view.seal && <span className={`tone-${view.seal.tone}`}>{view.seal.text}</span>}
      {view.doneNote && <span>{view.doneNote}</span>}
      {view.touched.length > 0 && (
        <span className="crit touched" title={`Escreveu, neste turno, onde o trabalho é medido:\n${view.touched.join('\n')}`}>
          ⚠ tocou a régua
        </span>
      )}
      {measure && !measure.measured && (
        <span
          className="crit unmeasured"
          title={`Os contratos de comportamento do dcode nunca rodaram contra a família ${measure.family || 'deste modelo'}: os critérios conferem o trabalho, e o modelo ninguém mediu.`}
        >
          ◌ sem medição
        </span>
      )}
    </div>
  );
}

/** What a window keeps of its conversation, and what it does with it. */
export interface PaneProps {
  row: Row;
  /** Its events, once followed; an ended conversation has none here. */
  session: SessionView | undefined;
  view: PaneView;
  focused: boolean;
  now: number;
  verbTick: number;
  draft: string;
  caret?: number;
  measure: Measure | null;
  /** Clicked into: the window is the person's now, and what it did is seen. */
  onFocus: () => void;
  onMaximize: () => void;
  /** Null while it runs or waits: what works keeps its place. */
  onClose: (() => void) | null;
  onDraft: (text: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
  onAnswer: (approvalId: string, decision: string) => void;
  /** An ended conversation, written to: it continues in a new session. */
  onContinue: (text: string) => void;
  onModel: (anchor: HTMLElement) => void;
  onMissing: (what: string) => void;
}

/**
 * One conversation, whole, in its window on the grid: its seal and criteria,
 * its flow and its own field. Each window scrolls, drafts and answers on its
 * own — the person works in any of them without opening it.
 */
export function Pane(p: PaneProps) {
  const { row, session, view: v } = p;
  const ended = row.state === 'recorded' && !session;
  const waiting = session?.state === 'blocked' || row.state === 'blocked';
  return (
    <section
      className={`pane${p.focused ? ' focused' : ''}${waiting ? ' waiting' : ''}`}
      data-pane-id={row.id}
      aria-label={row.title}
      onMouseDown={p.onFocus}
    >
      <div className="pane-head" onDoubleClick={p.onMaximize}>
        <span className={`pane-glyph tone-${v.tone}${v.live ? ' dc-breath' : ''}`}>{v.glyph}</span>
        <span className="pane-title" title={row.title}>
          {row.title}
        </span>
        {row.model && (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => p.onModel(e.currentTarget)}>
            {session?.info?.model ?? row.model} ⌄
          </button>
        )}
        <button type="button" className="icon-btn" aria-label={`Maximizar ${row.title}`} title="Maximizar (⌘↵)" onClick={p.onMaximize}>
          ⤢
        </button>
        {p.onClose && (
          <button type="button" className="icon-btn" aria-label={`Fechar ${row.title}`} title="Fechar a janela" onClick={p.onClose}>
            ×
          </button>
        )}
      </div>
      <DoneRow view={v} measure={p.measure} />
      <div className="pane-body">
        {ended ? (
          <>
            <div className="pane-ended">
              Terminada · {row.turns ?? 0} {row.turns === 1 ? 'turno' : 'turnos'}. Escrever continua a conversa numa sessão nova, com o
              histórico.
            </div>
            <div className="composer-wrap compact">
              <div className="composer">
                <Field
                  text={p.draft}
                  placeholder="Escreva para continuar esta conversa"
                  caret={p.caret}
                  onText={p.onDraft}
                  onSubmit={() => p.draft.trim() && p.onContinue(p.draft)}
                />
              </div>
            </div>
          </>
        ) : (
          <ConversationBody
            session={session ?? emptySession(row.id)}
            now={p.now}
            verbTick={p.verbTick}
            draft={p.draft}
            compact
            caret={p.caret}
            onDraft={p.onDraft}
            onAnswer={(decision) => session?.pendingApprovalId && p.onAnswer(session.pendingApprovalId, decision)}
            onSend={p.onSend}
            onStop={p.onStop}
            onMissing={p.onMissing}
          />
        )}
      </div>
      <div className="pane-foot">
        <span className="pane-where">{v.where}</span>
        <span className="spacer" />
        <span>{v.when}</span>
      </div>
    </section>
  );
}
