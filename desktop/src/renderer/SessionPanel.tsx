import { useLayoutEffect, useMemo, useRef } from 'react';
import { activity, flowBlocks } from '../state/flow';
import type { SessionView } from '../state/session';
import { sessionTitle } from '../state/sidebar';
import { Composer } from './Composer';
import { ActivityLine, FlowBlocks } from './Flow';

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

export function SessionPanel({
  session,
  now,
  verbTick,
  draft,
  onDraft,
  onAnswer,
  onSend,
  onStop,
  onModel,
  onMissing,
}: {
  session: SessionView;
  now: number;
  verbTick: number;
  draft: string;
  onDraft: (text: string) => void;
  onAnswer: (decision: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
  onModel: (anchor: HTMLElement) => void;
  onMissing: (what: string) => void;
}) {
  const blocks = useMemo(() => flowBlocks(session.entries), [session.entries]);
  const act = useMemo(() => activity(session.entries), [session.entries]);
  const scroller = useRef<HTMLDivElement>(null);
  const following = useRef(true);

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
    <section className="panel" aria-label={sessionTitle(session)}>
      <header className="panel-head drag-region">
        <span className="panel-title">{sessionTitle(session)}</span>
        <button type="button" className="panel-menu no-drag" aria-label="Menu da sessão" onClick={() => onMissing('O menu da sessão')}>
          ⌄
        </button>
        <span className="spacer" />
        <BranchChip session={session} />
      </header>
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
      <Composer session={session} text={draft} onText={onDraft} onSend={onSend} onStop={onStop} onModel={onModel} onMissing={onMissing} />
    </section>
  );
}
