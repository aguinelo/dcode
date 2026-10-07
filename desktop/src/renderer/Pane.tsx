import type * as P from '../protocol/generated';
import { boundaryLabel } from '../state/flow';
import type { PaneView } from '../state/pane';
import type { Row } from '../state/sidebar';
import { CHOICES } from './ApprovalCard';

/** What the harness measured: each criterion, else the seal, else what is known about done. */
export function DoneRow({ view }: { view: Pick<PaneView, 'lights' | 'seal' | 'doneNote'> }) {
  return (
    <div className="done-row">
      {view.lights.map((l) => (
        <span key={l.name} className={`crit ${l.state}`} title={l.state === 'unavailable' ? 'não pôde rodar' : undefined}>
          {l.state === 'met' ? '✓' : l.state === 'unmet' ? '✗' : '?'} {l.name}
        </span>
      ))}
      {view.seal && <span className={`tone-${view.seal.tone}`}>{view.seal.text}</span>}
      {view.doneNote && <span>{view.doneNote}</span>}
    </div>
  );
}

/** The question a session waits on, answered from its panel: 1 and 2 allow, ↵ denies (D22). */
function Ask({ request, onAnswer }: { request: P.ApprovalRequest; onAnswer: (decision: string) => void }) {
  return (
    <div className="ask" onClick={(e) => e.stopPropagation()}>
      <div className="ask-title">
        {request.command ? 'Rodar este comando?' : `Permitir ${request.tool}?`}{' '}
        <span className="tone-faint">pede {boundaryLabel(request.boundary_crossed)}</span>
      </div>
      {request.command && <div className="ask-what">{request.command}</div>}
      <div className="ask-actions">
        {CHOICES.map((c) => (
          <button key={c.key} type="button" className={`ask-btn${c.decision === 'deny' ? ' deny' : ''}`} onClick={() => onAnswer(c.decision)}>
            <span className="ask-key">{c.decision === 'deny' ? '↵' : c.key}</span>
            {c.label}
          </button>
        ))}
      </div>
    </div>
  );
}

export function Pane({
  row,
  view,
  focused,
  pinned,
  unseen,
  onFocus,
  onOpen,
  onPin,
  onSeen,
  onAnswer,
  onModel,
}: {
  row: Row;
  view: PaneView;
  focused: boolean;
  pinned: boolean;
  /** Finished since the person last looked: it can be dismissed without opening. */
  unseen: boolean;
  onFocus: () => void;
  onOpen: () => void;
  onPin: () => void;
  onSeen: () => void;
  onAnswer: (approvalId: string, decision: string) => void;
  onModel: (anchor: HTMLElement) => void;
}) {
  const v = view;
  return (
    <div
      className={`pane${focused ? ' focused' : ''}${v.approval ? ' waiting' : ''}`}
      data-pane-id={row.id}
      onClick={onFocus}
      onDoubleClick={onOpen}
    >
      <div className="pane-head">
        <span className={`pane-glyph tone-${v.tone}${v.live ? ' dc-breath' : ''}`}>{v.glyph}</span>
        <span className="pane-title" title={row.title}>
          {row.title}
        </span>
        {row.model && (
          <button
            type="button"
            className="chip"
            title="Continuar em outro modelo"
            onClick={(e) => {
              e.stopPropagation();
              onModel(e.currentTarget);
            }}
          >
            {row.model} ⌄
          </button>
        )}
        <button
          type="button"
          className={`icon-btn${pinned ? ' on' : ''}`}
          aria-label={pinned ? 'Soltar da grade' : 'Fixar na grade'}
          title={pinned ? 'Fixada: fica na grade mesmo parada' : 'Fixar na grade'}
          onClick={(e) => {
            e.stopPropagation();
            onPin();
          }}
        >
          {pinned ? '◉' : '○'}
        </button>
        <button
          type="button"
          className="icon-btn"
          aria-label={`Abrir ${row.title}`}
          title="Abrir a conversa (↵)"
          onClick={(e) => {
            e.stopPropagation();
            onOpen();
          }}
        >
          ⤢
        </button>
      </div>
      <DoneRow view={v} />
      <div className="pane-body">
        {v.lines.map((l, i) => {
          switch (l.kind) {
            case 'you':
              return (
                <div key={i} className="tail-you">
                  {l.text}
                </div>
              );
            case 'work':
              return (
                <div key={i} className="tail-work">
                  <span className={`tone-${l.tone}`}>{l.glyph}</span> {l.text}
                </div>
              );
            case 'said':
              return (
                <div key={i} className="tail-said">
                  {l.text}
                </div>
              );
            default:
              return (
                <div key={i} className={`tail-note tone-${l.tone}`}>
                  {l.text}
                </div>
              );
          }
        })}
        {v.approval && <Ask request={v.approval} onAnswer={(d) => onAnswer(v.approval?.approval_id ?? '', d)} />}
      </div>
      <div className="pane-foot">
        <span className="pane-where">{v.where}</span>
        <span className="spacer" />
        {unseen && (
          <button
            type="button"
            className="seen-btn"
            title="Tirar da grade sem abrir"
            onClick={(e) => {
              e.stopPropagation();
              onSeen();
            }}
          >
            dispensar
          </button>
        )}
        <span>{v.when}</span>
      </div>
    </div>
  );
}
