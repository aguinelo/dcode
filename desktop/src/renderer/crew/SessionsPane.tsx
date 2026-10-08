import { ChevronDown, ChevronRight, Plus, Search } from 'lucide-react';
import { useMemo, useState } from 'react';
import { groupCount, sessionGroups, stateWord } from '../../state/crew';
import { paneView, sealWord } from '../../state/pane';
import { lastEventText, searchSide } from '../../state/search';
import type { SessionsState } from '../../state/sessions';
import { basename, type ProjectView, type Row } from '../../state/sidebar';

function SessionItem({ row, sessions, now, on, onPick }: { row: Row; sessions: SessionsState; now: number; on: boolean; onPick: (r: Row) => void }) {
  const v = paneView(row, sessions.byId[row.id], now);
  const side = searchSide(row, now);
  const state = sessions.byId[row.id]?.state ?? row.state;
  const seal = state === 'running' || state === 'blocked' ? null : sealWord(row.verification);
  const last = lastEventText(row, now);
  return (
    <button type="button" className={`crew-item${on ? ' on' : ''}`} data-session-id={row.id} aria-current={on ? 'true' : undefined} onClick={() => onPick(row)}>
      <span className="crew-item-top">
        <span className={`crew-item-glyph tone-${v.tone}${v.live ? ' dc-breath' : ''}`}>{v.glyph}</span>
        <span className="crew-item-project mono">{basename(row.workspace) || '(sem projeto)'}</span>
        <span className={`crew-item-side tone-${side.tone}`}>{side.text}</span>
      </span>
      <span className="crew-item-title">{row.title}</span>
      <span className="crew-item-meta">
        {seal ? <span className={`tone-${seal.tone}`}>{seal.text}</span> : <span>{stateWord(state)}</span>}
        {last && <span className="crew-item-last">{last}</span>}
      </span>
    </button>
  );
}

/**
 * The left pane of the Painel: every conversation the daemon lists, searched
 * by title and project, grouped as active, by project, and older (D33).
 */
export function SessionsPane({
  projects,
  sessions,
  focusId,
  now,
  onPick,
  onStart,
}: {
  projects: readonly ProjectView[];
  sessions: SessionsState;
  focusId: string | null;
  now: number;
  onPick: (row: Row) => void;
  /** Opens a session in a known project, or from the folder picker with null. */
  onStart: (workspace: string | null) => void;
}) {
  const [query, setQuery] = useState('');
  const [olderOpen, setOlderOpen] = useState(false);
  const [menu, setMenu] = useState(false);
  const groups = useMemo(() => sessionGroups(projects, query, now), [projects, query, now]);
  const total = projects.reduce((n, p) => n + p.rows.length, 0);
  const known = projects.filter((p) => p.id);
  const start = (ws: string | null) => {
    setMenu(false);
    onStart(ws);
  };
  return (
    <section className="crew-sessions" aria-label="Sessões">
      <header className="crew-pane-head">
        <h2>Sessões</h2>
        <span className="crew-new">
          <button
            type="button"
            className="crew-primary"
            aria-expanded={known.length > 0 ? menu : undefined}
            onClick={() => (known.length > 0 ? setMenu((m) => !m) : start(null))}
          >
            <Plus size={13} aria-hidden /> Nova {known.length > 0 && <ChevronDown size={12} aria-hidden />}
          </button>
          {menu && (
            <span className="crew-new-menu" role="menu" onKeyDown={(e) => e.key === 'Escape' && setMenu(false)}>
              <span className="crew-new-title">Abrir em</span>
              {known.map((p) => (
                <button key={p.id} type="button" role="menuitem" title={p.id} onClick={() => start(p.id)}>
                  {p.label}
                </button>
              ))}
              <button type="button" role="menuitem" onClick={() => start(null)}>
                Outra pasta…
              </button>
            </span>
          )}
        </span>
      </header>
      <div className="crew-search">
        <Search size={13} className="crew-search-icon" aria-hidden />
        <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Procurar sessões…" aria-label="Procurar sessões" spellCheck={false} />
        <span className="crew-search-count">{groupCount(groups)}</span>
      </div>
      <div className="crew-sessions-list">
        {total === 0 && <p className="crew-quiet crew-pad">O daemon ainda não tem conversas. Abra uma sessão com “Nova”.</p>}
        {total > 0 && groups.length === 0 && <p className="crew-quiet crew-pad">Nenhuma sessão com “{query}”.</p>}
        {groups.map((g) => {
          const open = !g.collapsible || olderOpen || query.trim() !== '';
          return (
            <div key={g.key} className="crew-group">
              {g.collapsible ? (
                <button type="button" className="crew-group-head toggle" aria-expanded={open} onClick={() => setOlderOpen((o) => !o)}>
                  {open ? <ChevronDown size={12} aria-hidden /> : <ChevronRight size={12} aria-hidden />}
                  {g.label}
                  <span className="crew-count">{g.rows.length}</span>
                </button>
              ) : (
                <h3 className="crew-group-head">
                  {g.label}
                  <span className="crew-count">{g.rows.length}</span>
                </h3>
              )}
              {open &&
                g.rows.map((r) => <SessionItem key={r.id} row={r} sessions={sessions} now={now} on={r.id === focusId} onPick={onPick} />)}
            </div>
          );
        })}
      </div>
    </section>
  );
}
