import { useMemo, useRef } from 'react';
import { filesTouched } from '../../state/context';
import type { SessionView } from '../../state/session';
import type { Row } from '../../state/sidebar';
import { movedKey } from '../../state/git';
import { placeBy, usePopover } from '../popover';
import { useGit } from '../useGit';
import { GitChanges } from './GitChanges';
import { GitFacts } from './GitFacts';

const WIDTH = 460;

/**
 * Where a grid conversation works, opened from its branch chip or its foot
 * (D41): the same facts and changes Crew's right panel shows, in a card over
 * the window, so the grid's look stays as approved until someone asks.
 */
export function ContextCard({
  row,
  session,
  anchor,
  onClose,
}: {
  row: Row;
  session: SessionView | undefined;
  anchor: HTMLElement;
  onClose: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  const moved = movedKey(row, session);
  const { state, refresh } = useGit(row.id, moved);
  usePopover(box, anchor, onClose, '.ctx-file-row', state.kind);
  const wrote = useMemo(() => new Set(session ? filesTouched(session.entries).map((f) => f.path) : []), [session]);
  return (
    <div ref={box} className="ctx-card" role="dialog" aria-label={`Onde ${row.title} trabalha`} style={{ width: WIDTH, ...placeBy(anchor, WIDTH) }}>
      <div className="ctx-card-title">{row.title}</div>
      <GitFacts row={row} sessionState={session?.state} startedOn={session?.info?.branch ?? row.branch} state={state} />
      <div className="ctx-card-changes">
        <GitChanges conversationId={row.id} state={state} moved={moved} wrote={wrote} onRefresh={refresh} />
      </div>
    </div>
  );
}
