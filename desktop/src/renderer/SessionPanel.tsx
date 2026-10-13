import type { Measure } from '../state/models';
import { paneView, type PaneView } from '../state/pane';
import type { SessionView } from '../state/session';
import { rowOfSession, sessionTitle, type Row } from '../state/sidebar';
import { ConversationBody } from './ConversationBody';
import { DoneRow } from './Pane';

function BranchChip({ session }: { session: SessionView }) {
  const branch = session.info?.branch ?? '';
  // Nothing reported is drawn as nothing, the reading the TUI's bar makes:
  // "+0 −0" would claim no change where the tools simply said none.
  const changed = session.added > 0 || session.removed > 0;
  if (!branch && !changed) return null;
  return (
    <span className="branch-chip" title={changed ? 'linhas que as ferramentas reportaram nesta sessão' : undefined}>
      {branch && <span>{branch}</span>}
      {changed && <span className="tone-ok">+{session.added}</span>}
      {changed && <span className={session.removed > 0 ? 'tone-err' : 'tone-faint'}>−{session.removed}</span>}
    </span>
  );
}

export interface Peer {
  row: Row;
  view: PaneView;
}

/**
 * One conversation opened whole. Above it, the others on the grid, one click
 * away, so opening one never hides what the rest are doing.
 */
export function SessionPanel({
  session,
  row,
  peers,
  now,
  verbTick,
  draft,
  onDraft,
  onAnswer,
  onSend,
  onStop,
  onMissing,
  onGrid,
  onOpenPeer,
  onModel,
  measure,
  caret,
}: {
  session: SessionView;
  /** The conversation as the list has it, when the list has it yet. */
  row: Row | null;
  peers: Peer[];
  now: number;
  verbTick: number;
  draft: string;
  onDraft: (text: string) => void;
  onAnswer: (decision: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
  onMissing: (what: string) => void;
  onGrid: () => void;
  onOpenPeer: (id: string) => void;
  onModel: (anchor: HTMLElement) => void;
  /** What the daemon's list says of the family it runs on. */
  measure: Measure | null;
  /** Changes when the caret is asked into the field. */
  caret?: number;
}) {
  const known = row ?? rowOfSession(session);
  const proof = paneView(known, session, now);
  // A session that has said nothing yet still has the list's title — the one a
  // continuation inherits from the conversation it continues.
  const title = session.name || session.firstQuestion ? sessionTitle(session) : known.title;
  const model = session.info?.model ?? known.model;

  return (
    <section className="panel" aria-label={title}>
      <header className="panel-head">
        <button type="button" className="back-btn" onClick={onGrid} title="Voltar à grade">
          ▦ Grade <span className="back-key">esc</span>
        </button>
        <div className="peers">
          {peers.map((p) => (
            <button key={p.row.id} type="button" className="peer" title={p.row.title} onClick={() => onOpenPeer(p.row.id)}>
              <span className={`tone-${p.view.tone}${p.view.live ? ' dc-breath' : ''}`}>{p.view.glyph}</span>
              <span className="peer-title">{p.row.title}</span>
            </button>
          ))}
        </div>
      </header>
      <div className="panel-title-row">
        <span className={`pane-glyph tone-${proof.tone}${proof.live ? ' dc-breath' : ''}`}>{proof.glyph}</span>
        <span className="panel-title">{title}</span>
        {model && (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => onModel(e.currentTarget)}>
            {model} ⌄
          </button>
        )}
        <BranchChip session={session} />
      </div>
      <div className="proof">
        <DoneRow view={proof} measure={measure} />
      </div>
      <ConversationBody
        session={session}
        now={now}
        verbTick={verbTick}
        draft={draft}
        compact={false}
        caret={caret}
        onDraft={onDraft}
        onAnswer={onAnswer}
        onSend={onSend}
        onStop={onStop}
        onMissing={onMissing}
      />
    </section>
  );
}
