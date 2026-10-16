import { lazy, Suspense, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { imagesOfEntries } from '../state/attachments';
import { activity, flowBlocks } from '../state/flow';
import { pinnedPlan, withoutPinned } from '../state/plan';
import type { SessionView } from '../state/session';
import { Composer } from './Composer';
import { FlowBlocks } from './Flow';
import { PlanPin } from './PlanPin';
import { TurnActivity } from './TurnActivity';
import { useAttached } from './useAttachments';

// xterm.js is most of a megabyte: loaded the first time a terminal opens, not with the window.
const TerminalDock = lazy(() => import('./TerminalDock').then((m) => ({ default: m.TerminalDock })));

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
  onMode,
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
  onMode?: (anchor: HTMLElement) => void;
}) {
  // The running turn's plan stands above the flow, and the flow leaves it out (D57).
  const pin = useMemo(() => pinnedPlan(session), [session]);
  const blocks = useMemo(() => flowBlocks(withoutPinned(session.entries, pin)), [session.entries, pin]);
  const act = useMemo(() => activity(session.entries), [session.entries]);
  // The pictures this window sent, on the messages they went with (D46): the events carry none.
  const sent = useAttached().sent[session.id];
  const imagesAt = useMemo(() => imagesOfEntries(session.entries, sent), [session.entries, sent]);
  const scroller = useRef<HTMLDivElement>(null);
  const following = useRef(true);
  // The terminal docked below the flow (D37): one at a time, a new command a new shell.
  const [terminal, setTerminal] = useState<{ command: string; n: number } | null>(null);

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
      {pin && <PlanPin pin={pin} compact={compact} />}
      <div
        className="flow-scroll"
        ref={scroller}
        onScroll={(e) => {
          const el = e.currentTarget;
          following.current = el.scrollHeight - el.scrollTop - el.clientHeight <= FOLLOW_SLACK;
        }}
      >
        <div className="flow">
          <FlowBlocks blocks={blocks} now={now} onAnswer={onAnswer} imagesAt={imagesAt} onRun={(command) => setTerminal((t) => ({ command, n: (t?.n ?? 0) + 1 }))} />
          {session.state === 'running' && session.turn && (
            <TurnActivity session={session} activity={act} now={now} tick={verbTick} compact={compact} imagesAt={imagesAt} />
          )}
        </div>
      </div>
      {terminal && (
        <Suspense fallback={<div className={`term-dock${compact ? ' compact' : ''}`} />}>
          <TerminalDock
          key={terminal.n}
          sessionId={session.id}
          command={terminal.command}
          compact={compact}
          onSend={onSend}
          onClose={() => setTerminal(null)}
          />
        </Suspense>
      )}
      <Composer
        session={session}
        text={draft}
        compact={compact}
        caret={caret}
        onText={onDraft}
        onSend={onSend}
        onStop={onStop}
        onMissing={onMissing}
        onMode={onMode}
      />
    </>
  );
}
