import { useEffect, useState } from 'react';
import { MODES } from '../state/mode';
import { menuOf, type ModelOption } from '../state/models';
import { shortPath } from '../state/sidebar';
import type { Outcome, StartOptions } from './window';
import type { ModelsResponse } from '../protocol/generated';

/**
 * What a new session opens on, chosen before it opens (D38): a profile or
 * model from what the daemon lists for the project (`GET /v1/models`), and a
 * mode. Left alone, each stays the daemon's default — nothing is sent for it.
 */
export function StartForm({
  workspace,
  listModels,
  opening,
  onOpen,
  onBack,
}: {
  workspace: string;
  listModels: (workspace: string) => Promise<Outcome<ModelsResponse>>;
  /** A session is being opened from here. */
  opening: boolean;
  onOpen: (options: StartOptions) => void;
  onBack: () => void;
}) {
  const [models, setModels] = useState<{ ws: string; options: ModelOption[] | null; problem: string | null }>({ ws: workspace, options: null, problem: null });
  const [model, setModel] = useState('');
  const [mode, setMode] = useState('');
  useEffect(() => {
    let live = true;
    void listModels(workspace).then((got) => {
      if (!live) return;
      setModels(got.ok ? { ws: workspace, options: menuOf(got.value, null), problem: null } : { ws: workspace, options: null, problem: got.why });
    });
    return () => {
      live = false;
    };
  }, [listModels, workspace]);
  const listed = models.ws === workspace ? models : { options: null, problem: null };
  const fallback = listed.options?.find((o) => o.isDefault)?.name ?? '';
  const open = () => {
    const options: StartOptions = {};
    if (model) options.model = model;
    if (mode) options.mode = mode;
    onOpen(options);
  };
  return (
    <div className="start-form" aria-label={`Sessão nova em ${workspace}`}>
      <div className="start-where">
        <button type="button" className="start-back" onClick={onBack} disabled={opening}>
          ‹ projetos
        </button>
        <span className="mono" title={workspace}>
          {shortPath(workspace)}
        </span>
      </div>
      <label className="start-field">
        <span className="start-label">Modelo</span>
        {listed.problem ? (
          <span className="start-note tone-warn">{listed.problem}</span>
        ) : (
          <select value={model} onChange={(e) => setModel(e.target.value)} disabled={opening || listed.options === null} aria-label="Modelo">
            <option value="">{listed.options === null ? 'perguntando ao daemon…' : `padrão do projeto${fallback ? ` — ${fallback}` : ''}`}</option>
            {(listed.options ?? [])
              .filter((o) => !o.isDefault)
              .map((o) => (
                <option key={o.name} value={o.name} title={o.notice ?? undefined}>
                  {o.name} — {o.detail}
                  {o.measured ? '' : ' · sem medição'}
                </option>
              ))}
          </select>
        )}
      </label>
      <div className="start-field" role="radiogroup" aria-label="Modo">
        <span className="start-label">Modo</span>
        <span className="start-modes">
          {[{ mode: '', label: 'Padrão', about: 'o que o daemon abre' }, ...MODES].map((m) => (
            <button
              key={m.mode || 'default'}
              type="button"
              role="radio"
              aria-checked={mode === m.mode}
              className={`start-mode${mode === m.mode ? ' on' : ''}${m.mode === 'auto' ? ' tone-warn' : ''}`}
              title={m.about}
              disabled={opening}
              onClick={() => setMode(m.mode)}
            >
              {m.label}
            </button>
          ))}
        </span>
      </div>
      <button type="button" className="start-open" disabled={opening} onClick={open}>
        {opening ? 'Abrindo…' : 'Abrir sessão'}
      </button>
    </div>
  );
}
