import { useEffect, useState } from 'react';
import type { LoopRun, SpecFolder } from '../protocol/generated';
import { plural } from '../state/format';
import { loopArgsOf } from '../state/loop';
import { shortPath } from '../state/sidebar';
import { SheetView } from './CommandSheet';
import type { Outcome } from './window';

type Listed = { state: 'loading' } | { state: 'ok'; folders: SpecFolder[] } | { state: 'failed'; why: string };

/** What a folder declares, as the daemon said it: never a guess about what it will need. */
function declared(f: SpecFolder): string {
  if (f.error) return f.error;
  return f.criteria === 0 ? '0 · qualifica antes' : plural(f.criteria, 'critério', 'critérios');
}

/**
 * "Começar loop" (D55, Q6 C): a folder the daemon lists, or a goal, and what
 * to protect — sent as the argument a person would type after `/loop`, so the
 * field and the sheet speak one grammar, the daemon's. What a folder declares
 * is what the daemon says (`GET /v1/specs?measure=false`), read, not run.
 */
export function LoopSheet({
  workspace,
  live,
  listSpecs,
  onStart,
  onClose,
}: {
  workspace: string;
  /** The loop alive in this project, if one is (Q7). */
  live: LoopRun | null;
  listSpecs: (workspace: string) => Promise<Outcome<SpecFolder[]>>;
  /** Starts the loop: done, or why not, to be said here. */
  onStart: (args: string) => Promise<Outcome<unknown>>;
  onClose: () => void;
}) {
  const [listed, setListed] = useState<Listed>({ state: 'loading' });
  const [kind, setKind] = useState<'folder' | 'goal'>('folder');
  const [folder, setFolder] = useState('');
  const [goal, setGoal] = useState('');
  const [protect, setProtect] = useState('');
  const [why, setWhy] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);

  useEffect(() => {
    let on = true;
    void listSpecs(workspace).then((got) => {
      if (on) setListed(got.ok ? { state: 'ok', folders: got.value } : { state: 'failed', why: got.why });
    });
    return () => {
      on = false;
    };
  }, [listSpecs, workspace]);

  const args = loopArgsOf({ target: kind === 'folder' ? folder : goal, protect: protect.split(/[\s,]+/) });
  const start = async () => {
    if (!args.ok || starting) return setWhy(args.ok ? null : args.why);
    setStarting(true);
    const out = await onStart(args.args);
    setStarting(false);
    if (out.ok) onClose();
    else setWhy(out.why);
  };

  return (
    <SheetView title="Começar loop" note={shortPath(workspace, 60)} onClose={onClose}>
      <div
        className="start-form loop-sheet"
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            void start();
          }
        }}
      >
        {live && (
          <p className="sheet-empty tone-warn">
            Já há um loop vivo neste projeto ({live.goal || live.specs[live.current]?.path || live.state}): o daemon recusa outro até ele parar.
          </p>
        )}
        <div className="start-modes" role="radiogroup" aria-label="O que o loop faz">
          {(['folder', 'goal'] as const).map((k) => (
            <button key={k} type="button" role="radio" aria-checked={kind === k} className={`start-mode${kind === k ? ' on' : ''}`} onClick={() => setKind(k)}>
              {k === 'folder' ? 'Uma pasta' : 'Um objetivo'}
            </button>
          ))}
        </div>
        {kind === 'folder' ? (
          <div className="start-field">
            <span className="start-label">Pastas · critérios declarados</span>
            {listed.state === 'loading' && <span className="sheet-empty">Perguntando ao daemon quais pastas de spec este projeto tem…</span>}
            {listed.state === 'failed' && <span className="sheet-empty tone-err">{listed.why}</span>}
            {listed.state === 'ok' && listed.folders.length === 0 && (
              <span className="sheet-empty">O daemon não achou pasta de spec neste projeto. Escreva um objetivo, ou o caminho de uma pasta no campo com /loop.</span>
            )}
            {listed.state === 'ok' && listed.folders.length > 0 && (
              <div className="loop-folders" role="radiogroup" aria-label="Pastas">
                {listed.folders.map((f) => (
                  <button
                    key={f.path}
                    type="button"
                    role="radio"
                    aria-checked={folder === f.path}
                    className={`loop-folder${folder === f.path ? ' on' : ''}`}
                    onClick={() => setFolder(f.path)}
                  >
                    <span className="mono">{f.path}</span>
                    <span className={f.error ? 'tone-err' : 'tone-dim'}>{declared(f)}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <label className="start-field">
            <span className="start-label">Objetivo</span>
            <input className="loop-input" value={goal} autoFocus placeholder="implemente as specs pendentes" onChange={(e) => setGoal(e.target.value)} />
          </label>
        )}
        <label className="start-field">
          <span className="start-label">Protegido · o done.toml da pasta soma, nunca tira</span>
          <input className="loop-input mono" value={protect} placeholder="tests/**" onChange={(e) => setProtect(e.target.value)} />
        </label>
        <span className="loop-preview mono" title="O que vai ao daemon, como no campo">
          /loop {args.ok ? args.args : '…'}
        </span>
        {why && <p className="sheet-empty tone-err">{why}</p>}
        <div className="loop-sheet-actions">
          <button type="button" className="sheet-close" onClick={onClose}>
            Cancelar
          </button>
          <button type="button" className="start-open" disabled={!args.ok || starting} onClick={() => void start()}>
            {starting ? 'Começando…' : 'Começar'} <span className="sheet-key">⌘↵</span>
          </button>
        </div>
      </div>
    </SheetView>
  );
}
