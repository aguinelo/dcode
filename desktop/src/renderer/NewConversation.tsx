import { useState } from 'react';
import type { ModelsResponse } from '../protocol/generated';
import { shortPath } from '../state/sidebar';
import { StartForm } from './StartForm';
import type { Outcome, StartOptions } from './window';

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
  onLoop,
  onClose,
  listModels,
  pickFolder,
}: {
  places: readonly Place[];
  focused: boolean;
  /** Where a session is being opened, while it is. */
  opening: string | null;
  onFocus: () => void;
  onPlace: (workspace: string, options?: StartOptions) => void;
  onOtherFolder: () => void;
  /** "Começar loop…" in the project chosen (D55). */
  onLoop: (workspace: string) => void;
  onClose: () => void;
  /** What the daemon lists for a project, for choosing the model before opening (D38). */
  listModels: (workspace: string) => Promise<Outcome<ModelsResponse>>;
  pickFolder: () => Promise<string | null>;
}) {
  // With "modelo e modo" on, choosing a place goes to the form instead of opening there.
  const [choosing, setChoosing] = useState(false);
  // With "loop" on, choosing a place opens the loop's sheet there instead (D55).
  const [looping, setLooping] = useState(false);
  const [where, setWhere] = useState<string | null>(null);
  const pick = (ws: string) => (looping ? onLoop(ws) : choosing ? setWhere(ws) : onPlace(ws));
  const other = () => {
    if (!choosing && !looping) return onOtherFolder();
    void pickFolder().then((ws) => ws && pick(ws));
  };
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
        {where ? (
          <StartForm workspace={where} listModels={listModels} opening={opening !== null} onOpen={(o) => onPlace(where, o)} onBack={() => setWhere(null)} />
        ) : (
          <>
            <span className="new-label">
              Em qual projeto?
              <button
                type="button"
                className={`new-choose${choosing ? ' on' : ''}`}
                aria-pressed={choosing}
                title="Escolher o modelo e o modo antes de abrir"
                onClick={() => (setChoosing((c) => !c), setLooping(false))}
              >
                {choosing ? '✓ ' : '+ '}modelo e modo
              </button>
              <button
                type="button"
                className={`new-choose new-loop${looping ? ' on' : ''}`}
                aria-pressed={looping}
                title="Começar um loop no projeto escolhido: uma pasta de spec ou um objetivo"
                onClick={() => (setLooping((l) => !l), setChoosing(false))}
              >
                {looping ? '✓ ' : '↻ '}loop
              </button>
            </span>
            <div className="new-places">
              {places.slice(0, OFFERED).map((p, i) => (
                <button
                  key={p.id}
                  type="button"
                  className="place"
                  title={p.id}
                  disabled={opening !== null}
                  autoFocus={focused && i === 0}
                  onClick={() => pick(p.id)}
                >
                  <span className="place-label">{p.label}</span>
                  <span className="place-path mono">{shortPath(p.id)}</span>
                </button>
              ))}
              <button type="button" className="place other" disabled={opening !== null} onClick={other}>
                <span className="place-label">Outra pasta…</span>
              </button>
            </div>
          </>
        )}
      </div>
      <div className="pane-foot">
        <span className="pane-where">
          {opening
            ? `abrindo em ${opening}…`
            : looping
              ? 'escolher abre o começar loop'
              : choosing && !where
                ? 'escolher leva ao modelo e ao modo'
                : 'escolher abre a sessão aqui'}
        </span>
      </div>
    </section>
  );
}
