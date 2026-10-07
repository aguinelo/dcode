import type { Attention } from '../state/attention';
import { gridShape, type GridPlan } from '../state/grid';
import { paneView } from '../state/pane';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import { Pane } from './Pane';

/**
 * The stage when no conversation is open: every conversation that wants
 * attention, at once, in places that do not move while the work does.
 */
export function Grid({
  plan,
  rows,
  sessions,
  attention,
  focus,
  now,
  onFocus,
  onOpen,
  onPin,
  onSeen,
  onAnswer,
  onModel,
  onNew,
  onSearch,
}: {
  plan: GridPlan;
  rows: ReadonlyMap<string, Row>;
  sessions: SessionsState;
  attention: Attention;
  focus: number;
  now: number;
  onFocus: (index: number) => void;
  onOpen: (id: string) => void;
  onPin: (id: string) => void;
  onSeen: (id: string) => void;
  onAnswer: (sessionId: string, approvalId: string, decision: string) => void;
  onModel: (row: Row, anchor: HTMLElement) => void;
  onNew: () => void;
  onSearch: () => void;
}) {
  if (plan.ids.length === 0) {
    return (
      <div className="grid-empty">
        <strong>Nada pedindo sua atenção agora.</strong>
        <span>O que rodar, esperar você ou terminar aparece aqui sozinho.</span>
        <span style={{ display: 'flex', gap: 8 }}>
          <button type="button" className="nav-item" onClick={onNew}>
            + Nova sessão <span className="nav-key">⌘N</span>
          </button>
          <button type="button" className="nav-item" onClick={onSearch}>
            ⌕ Procurar <span className="nav-key">⌘K</span>
          </button>
        </span>
      </div>
    );
  }
  const shape = gridShape(plan.ids.length);
  return (
    <div
      className="grid"
      style={{ gridTemplateColumns: `repeat(${shape.cols}, minmax(0, 1fr))`, gridTemplateRows: `repeat(${shape.rows}, minmax(0, 1fr))` }}
    >
      {plan.ids.map((id, i) => {
        const row = rows.get(id);
        if (!row) return null;
        return (
          <Pane
            key={id}
            row={row}
            view={paneView(row, sessions.byId[id], now)}
            focused={i === focus}
            pinned={attention.pinned.includes(id)}
            unseen={plan.claims[id] === 'unseen'}
            onFocus={() => onFocus(i)}
            onOpen={() => onOpen(id)}
            onPin={() => onPin(id)}
            onSeen={() => onSeen(id)}
            onAnswer={(approvalId, decision) => onAnswer(id, approvalId, decision)}
            onModel={(anchor) => onModel(row, anchor)}
          />
        );
      })}
    </div>
  );
}
