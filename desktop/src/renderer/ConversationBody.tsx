import { useLayoutEffect, useMemo, useRef } from 'react';
import { activity, flowBlocks } from '../state/flow';
import type { SessionView } from '../state/session';
import { Composer } from './Composer';
import { ActivityLine, FlowBlocks } from './Flow';

/** How close to the end counts as "at the end" for auto-follow, in pixels. */
const FOLLOW_SLACK = 4;

/**
 * A conversation's flow and its field: what a grid window shows and what an
 * opened conversation shows, the same either way. Each keeps its own scroll.
 */
export function ConversationBody({
  session,
  now,
  verbTick,
  draft,
  compact,
  caret,
  onDraft,
  onAnswer,
  onSend,
  onStop,
  onMissing,
}: {
  session: SessionView;
  now: number;
  verbTick: number;
  draft: string;
  /** A window on the grid: the field without its extras, the flow in a smaller hand. */
  compact: boolean;
  /** Changes when the caret is asked into this field. */
  caret?: number;
  onDraft: (text: string) => void;
  onAnswer: (decision: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
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
    <>
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
      <Composer
        session={session}
        text={draft}
        compact={compact}
        caret={caret}
        onText={onDraft}
        onSend={onSend}
        onStop={onStop}
        onMissing={onMissing}
      />
    </>
  );
}
