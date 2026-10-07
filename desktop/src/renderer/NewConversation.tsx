import { shortPath } from '../state/sidebar';

/** A project a new session can open in. */
export interface Place {
  /** The workspace path. */
  id: string;
  label: string;
}

/** How many projects the window offers before "Outra pasta…": the most recently active. */
const OFFERED = 9;

/**
 * A window on the grid for a conversation about to start: it asks where.
 * Choosing opens the session there, and the window becomes that conversation,
 * in the same place, with the caret in its field.
 */
export function NewConversation({
  places,
  focused,
  opening,
  onFocus,
  onPlace,
  onOtherFolder,
  onClose,
}: {
  places: readonly Place[];
  focused: boolean;
  /** Where a session is being opened, while it is. */
  opening: string | null;
  onFocus: () => void;
  onPlace: (workspace: string) => void;
  onOtherFolder: () => void;
  onClose: () => void;
}) {
  return (
    <section className={`pane new-pane${focused ? ' focused' : ''}`} aria-label="Nova sessão" onMouseDown={onFocus}>
      <div className="pane-head">
        <span className="pane-glyph tone-accent">+</span>
        <span className="pane-title">Nova sessão</span>
        <button type="button" className="icon-btn" aria-label="Fechar a nova sessão" title="Fechar a janela" onClick={onClose}>
          ×
        </button>
      </div>
      <div className="pane-body new-body">
        <span className="new-label">Em qual projeto?</span>
        <div className="new-places">
          {places.slice(0, OFFERED).map((p, i) => (
            <button
              key={p.id}
              type="button"
              className="place"
              title={p.id}
              disabled={opening !== null}
              autoFocus={focused && i === 0}
              onClick={() => onPlace(p.id)}
            >
              <span className="place-label">{p.label}</span>
              <span className="place-path mono">{shortPath(p.id)}</span>
            </button>
          ))}
          <button type="button" className="place other" disabled={opening !== null} onClick={onOtherFolder}>
            <span className="place-label">Outra pasta…</span>
          </button>
        </div>
      </div>
      <div className="pane-foot">
        <span className="pane-where">{opening ? `abrindo em ${opening}…` : 'escolher abre a sessão aqui'}</span>
      </div>
    </section>
  );
}
