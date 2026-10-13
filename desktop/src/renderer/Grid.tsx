import { gridShape, type GridPlan } from '../state/grid';
import type { History } from '../state/history';
import type { Measure } from '../state/models';
import { paneView } from '../state/pane';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import { MascotFigure } from './Mascot';
import { NewConversation, type Place } from './NewConversation';
import { Pane } from './Pane';
import type { ModelsResponse } from '../protocol/generated';
import type { Outcome, StartOptions } from './window';

/** What the grid's windows do, by the conversation each holds. */
export interface WindowHandlers {
  focus: (id: string) => void;
  maximize: (id: string) => void;
  /** Null for one that cannot leave: it runs or waits. */
  closer: (id: string) => (() => void) | null;
  draft: (id: string, text: string) => void;
  send: (id: string, text: string) => void;
  stop: (id: string) => void;
  answer: (id: string, approvalId: string, decision: string) => void;
  continueIn: (row: Row, text: string) => void;
  model: (row: Row, anchor: HTMLElement) => void;
  mode: (id: string, anchor: HTMLElement) => void;
  actions: (id: string, anchor: HTMLElement) => void;
  missing: (what: string) => void;
  /** An ended conversation's history, read from its record (D42); undefined until asked for. */
  history: (id: string) => History | undefined;
  retryHistory: (id: string) => void;
}

/** What the windows for new sessions do, by their key. */
export interface ChooserHandlers {
  place: (key: string, workspace: string, options?: StartOptions) => void;
  otherFolder: (key: string) => void;
  listModels: (workspace: string) => Promise<Outcome<ModelsResponse>>;
  pickFolder: () => Promise<string | null>;
  close: (key: string) => void;
}

/**
 * The stage when no conversation is maximized: every conversation the person
 * keeps or that wants attention, each whole in its own window, in places that
 * do not move while the work does — and the windows of sessions about to open.
 */
export function Grid({
  plan,
  choosers,
  opening,
  places,
  rows,
  sessions,
  focusId,
  caret,
  drafts,
  now,
  verbTick,
  measureFor,
  on,
  chooser,
  onNew,
  onSearch,
}: {
  plan: GridPlan;
  /** Keys of the windows asking where a new session opens. */
  choosers: readonly string[];
  /** For each chooser opening a session, where. */
  opening: Readonly<Record<string, string>>;
  places: readonly Place[];
  rows: ReadonlyMap<string, Row>;
  sessions: SessionsState;
  focusId: string | null;
  /** The window the caret was last asked into, and how many times. */
  caret: { id: string; n: number } | null;
  drafts: Readonly<Record<string, string>>;
  now: number;
  verbTick: number;
  measureFor: (row: Row) => Measure | null;
  on: WindowHandlers;
  chooser: ChooserHandlers;
  onNew: () => void;
  onSearch: () => void;
}) {
  const total = plan.ids.length + choosers.length;
  if (total === 0) {
    return (
      <div className="grid-empty">
        <MascotFigure size={56} />
        <strong>Nada pedindo sua atenção agora.</strong>
        <span>O que você abrir fica aqui, cada conversa na sua janela; o que rodar, esperar você ou terminar aparece sozinho.</span>
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
  const shape = gridShape(total);
  return (
    <div
      className="grid"
      style={{ gridTemplateColumns: `repeat(${shape.cols}, minmax(0, 1fr))`, gridTemplateRows: `repeat(${shape.rows}, minmax(0, 1fr))` }}
    >
      {plan.ids.map((id) => {
        const row = rows.get(id);
        if (!row) return null;
        const session = sessions.byId[id];
        return (
          <Pane
            key={id}
            row={row}
            session={session}
            view={paneView(row, session, now)}
            focused={id === focusId}
            now={now}
            verbTick={verbTick}
            draft={drafts[id] ?? ''}
            caret={caret?.id === id ? caret.n : undefined}
            measure={measureFor(row)}
            onFocus={() => on.focus(id)}
            onMaximize={() => on.maximize(id)}
            onClose={on.closer(id)}
            onDraft={(text) => on.draft(id, text)}
            onSend={(text) => on.send(id, text)}
            onStop={() => on.stop(id)}
            onAnswer={(approvalId, decision) => on.answer(id, approvalId, decision)}
            onContinue={(text) => on.continueIn(row, text)}
            history={on.history(id)}
            onRetryHistory={() => on.retryHistory(id)}
            onModel={(anchor) => on.model(row, anchor)}
            onMode={(anchor) => on.mode(row.id, anchor)}
            onActions={(anchor) => on.actions(row.id, anchor)}
            onMissing={on.missing}
          />
        );
      })}
      {choosers.map((key) => (
        <NewConversation
          key={key}
          places={places}
          focused={key === focusId}
          opening={opening[key] ?? null}
          onFocus={() => on.focus(key)}
          onPlace={(workspace, options) => chooser.place(key, workspace, options)}
          onOtherFolder={() => chooser.otherFolder(key)}
          onClose={() => chooser.close(key)}
          listModels={chooser.listModels}
          pickFolder={chooser.pickFolder}
        />
      ))}
    </div>
  );
}
