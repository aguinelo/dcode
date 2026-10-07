import { useLayoutEffect, useMemo, useRef } from 'react';
import { activity, flowBlocks } from '../state/flow';
import { paneView, type PaneView } from '../state/pane';
import type { SessionView } from '../state/session';
import { rowOfSession, sessionTitle, type Row } from '../state/sidebar';
import { Composer } from './Composer';
import { ActivityLine, FlowBlocks } from './Flow';
import type { Measure } from '../state/models';
import { DoneRow } from './Pane';

/** How close to the end counts as "at the end" for auto-follow, in pixels. */
const FOLLOW_SLACK = 4;

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
}) {
  const blocks = useMemo(() => flowBlocks(session.entries), [session.entries]);
  const act = useMemo(() => activity(session.entries), [session.entries]);
  const scroller = useRef<HTMLDivElement>(null);
  const following = useRef(true);
  const known = row ?? rowOfSession(session);
  const proof = paneView(known, session, now);
  // A session that has said nothing yet still has the list's title — the one a
  // continuation inherits from the conversation it continues.
  const title = session.name || session.firstQuestion ? sessionTitle(session) : known.title;
  const model = session.info?.model ?? known.model;

  // Auto-follow: new content keeps the end in view unless the person scrolled
  // up; coming back to the end turns it on again. Another session opens at
  // its end.
  useLayoutEffect(() => {
    following.current = true;
  }, [session.id]);
  useLayoutEffect(() => {
    const el = scroller.current;
    if (el && following.current) el.scrollTop = el.scrollHeight;
  }, [blocks, session.id]);

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
      <div
        className="flow-scroll"
        ref={scroller}
        onScroll={(e) => {
          const el = e.currentTarget;
          following.current = el.scrollHeight - el.scrollTop - el.clientHeight <= FOLLOW_SLACK;
        }}
      >
        <div className="flow">
          <FlowBlocks blocks={blocks} now={now} onAnswer={onAnswer} />
          {session.state === 'running' && session.turn && (
            <ActivityLine activity={act} startedAt={session.turn.startedAt} now={now} tick={verbTick} />
          )}
        </div>
      </div>
      <Composer session={session} text={draft} onText={onDraft} onSend={onSend} onStop={onStop} onMissing={onMissing} />
    </section>
  );
}
