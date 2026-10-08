import { ChevronDown, FileDiff, GitBranch } from 'lucide-react';
import { useMemo, useState } from 'react';
import { filesTouched } from '../../state/context';
import { contextPercent, stateWord } from '../../state/crew';
import type { Measure } from '../../state/models';
import type { PaneView } from '../../state/pane';
import type { SessionView } from '../../state/session';
import { shortPath, type Row } from '../../state/sidebar';
import { DoneRow } from '../Pane';

type Tab = 'mudancas' | 'pronto' | 'sessao';

function Changes({ session, row }: { session: SessionView | undefined; row: Row }) {
  const files = useMemo(() => (session ? filesTouched(session.entries) : []), [session]);
  if (!session) {
    return (
      <p className="crew-quiet">
        {row.state === 'recorded'
          ? 'Uma conversa terminada não traz os eventos dela: os arquivos aparecem quando ela continua numa sessão nova.'
          : 'Os eventos desta conversa ainda estão chegando.'}
      </p>
    );
  }
  if (files.length === 0) return <p className="crew-quiet">Nenhum arquivo escrito ainda nesta sessão.</p>;
  const added = files.reduce((n, f) => n + f.added, 0);
  const removed = files.reduce((n, f) => n + f.removed, 0);
  return (
    <>
      <div className="crew-tab-sum">
        <span>
          {files.length} {files.length === 1 ? 'arquivo mudado' : 'arquivos mudados'}
        </span>
        <span className="mono">
          <span className="tone-ok">+{added}</span> <span className="tone-err">−{removed}</span>
        </span>
      </div>
      <ul className="crew-files">
        {files.map((f) => (
          <li key={f.path} className={f.failed ? 'failed' : undefined} title={f.path}>
            <FileDiff size={12} className="crew-file-icon" aria-hidden />
            <span className="crew-file mono">{f.path}</span>
            {f.failed ? (
              <span className="tone-err">falhou</span>
            ) : (
              <span className="crew-diff mono">
                <span className="tone-ok">+{f.added}</span> <span className="tone-err">−{f.removed}</span>
              </span>
            )}
          </li>
        ))}
      </ul>
      <p className="crew-quiet crew-foot-note">Lido das chamadas de escrita e das linhas que as ferramentas reportaram (D8).</p>
    </>
  );
}

function Done({ view, measure }: { view: PaneView; measure: Measure | null }) {
  return (
    <>
      <DoneRow view={view} measure={measure} />
      {view.touched.length > 0 && (
        <div className="crew-touched">
          <span className="tone-warn">Escreveu, no último turno, onde o trabalho é medido:</span>
          <ul className="crew-files">
            {view.touched.map((p) => (
              <li key={p}>
                <span className="crew-file mono">{p}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <p className="crew-quiet crew-foot-note">O selo é o que os critérios conferiram ao fim do turno, não o que o modelo disse (D13).</p>
    </>
  );
}

function Facts({ row, session, onModel }: { row: Row; session: SessionView | undefined; onModel: (anchor: HTMLElement) => void }) {
  const info = session?.info;
  const pct = contextPercent(session);
  const branch = info?.branch ?? row.branch;
  const model = info?.model ?? row.model;
  return (
    <dl className="crew-facts">
      <dt>projeto</dt>
      <dd className="mono" title={row.workspace}>
        {shortPath(row.workspace, 30)}
      </dd>
      <dt>branch</dt>
      <dd className="mono">
        {branch ? (
          <>
            <GitBranch size={11} aria-hidden /> {branch}
          </>
        ) : (
          '—'
        )}
      </dd>
      <dt>modelo</dt>
      <dd>
        {model ? (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => onModel(e.currentTarget)}>
            {model} <ChevronDown size={11} aria-hidden />
          </button>
        ) : (
          '—'
        )}
      </dd>
      <dt>estado</dt>
      <dd>{stateWord(session?.state ?? row.state)}</dd>
      {session?.sandbox && (
        <>
          <dt>modo</dt>
          <dd className="mono">
            {session.sandbox} · {session.mode}
          </dd>
        </>
      )}
      <dt>turnos</dt>
      <dd>{session?.turns ?? row.turns ?? 0}</dd>
      <dt>contexto</dt>
      <dd>{pct !== null ? `${pct}%` : <span className="tone-faint">medido ao fim do primeiro turno</span>}</dd>
    </dl>
  );
}

/**
 * The right pane of the Painel: what the conversation in focus established,
 * in tabs — the files it changed, its seal and criteria, and where it runs.
 * Only what the daemon carries; nothing here stands in for a pull request.
 */
export function ContextPane({
  row,
  view,
  session,
  measure,
  onModel,
}: {
  row: Row | null;
  view: PaneView | null;
  session: SessionView | undefined;
  measure: Measure | null;
  onModel: (row: Row, anchor: HTMLElement) => void;
}) {
  const [tab, setTab] = useState<Tab>('mudancas');
  const files = useMemo(() => (session ? filesTouched(session.entries).length : null), [session]);
  const tabs: { id: Tab; label: string; count?: string }[] = [
    { id: 'mudancas', label: 'Mudanças', count: files !== null && files > 0 ? String(files) : undefined },
    { id: 'pronto', label: 'Pronto' },
    { id: 'sessao', label: 'Sessão' },
  ];
  return (
    <aside className="crew-context" aria-label="Contexto da conversa">
      <div className="crew-tabs" role="tablist">
        {tabs.map((t) => (
          <button key={t.id} type="button" role="tab" aria-selected={t.id === tab} className={t.id === tab ? 'on' : ''} onClick={() => setTab(t.id)}>
            {t.label}
            {t.count && <span className="crew-count">{t.count}</span>}
          </button>
        ))}
      </div>
      <div className="crew-tab-body" role="tabpanel">
        {!row || !view ? (
          <p className="crew-quiet">Nenhuma conversa em foco.</p>
        ) : tab === 'mudancas' ? (
          <Changes session={session} row={row} />
        ) : tab === 'pronto' ? (
          <Done view={view} measure={measure} />
        ) : (
          <Facts row={row} session={session} onModel={(anchor) => onModel(row, anchor)} />
        )}
      </div>
    </aside>
  );
}
