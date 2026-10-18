import { ChevronDown, ChevronRight, Plus, Search } from 'lucide-react';
import { useMemo, useState } from 'react';
import { groupCount, sessionGroups, stateWord } from '../../state/crew';
import { paneView, sealWord } from '../../state/pane';
import { lastEventText, searchSide } from '../../state/search';
import type { SessionsState } from '../../state/sessions';
import { basename, type ProjectView, type Row } from '../../state/sidebar';
import type { ModelsResponse } from '../../protocol/generated';
import { MascotGlyph } from '../Mascot';
import { StartForm } from '../StartForm';
import type { Outcome, StartOptions } from '../window';

function SessionItem({ row, sessions, now, on, depth, onPick }: { row: Row; sessions: SessionsState; now: number; on: boolean; depth: number; onPick: (r: Row) => void }) {
  const v = paneView(row, sessions.byId[row.id], now);
  const side = searchSide(row, now);
  const state = sessions.byId[row.id]?.state ?? row.state;
  const seal = state === 'running' || state === 'blocked' ? null : sealWord(row.verification);
  const last = lastEventText(row, now);
  // A child of an orchestrator under its parent (D66): drawn in by how deep it is.
  return (
    <button
      type="button"
      className={`crew-item${depth > 0 ? ' crew-child' : ''}${on ? ' on' : ''}`}
      data-session-id={row.id}
      data-state={row.state}
      data-depth={depth > 0 ? depth : undefined}
      style={depth > 0 ? { paddingLeft: 10 + depth * 14 } : undefined}
      aria-current={on ? 'true' : undefined}
      onClick={() => onPick(row)}
    >
      <span className="crew-item-top">
        <MascotGlyph state={v.mascot} live={v.live} size={12} className={`crew-item-glyph tone-${v.tone}`} />
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
  onLoop,
  onOrchestrate,
  listModels,
  pickFolder,
}: {
  projects: readonly ProjectView[];
  sessions: SessionsState;
  focusId: string | null;
  now: number;
  onPick: (row: Row) => void;
  /** Opens a session in a known project, or from the folder picker with null. */
  onStart: (workspace: string | null, options?: StartOptions) => void;
  /** "Começar loop…" in the project chosen (D55, D63). */
  onLoop: (workspace: string) => void;
  /** Opens the project's orchestrator, in a known project or from the folder picker with null (D66). */
  onOrchestrate: (workspace: string | null) => void;
  /** For choosing the model and mode before opening (D38). */
  listModels: (workspace: string) => Promise<Outcome<ModelsResponse>>;
  pickFolder: () => Promise<string | null>;
}) {
  const [query, setQuery] = useState('');
  const [olderOpen, setOlderOpen] = useState(false);
  const [menu, setMenu] = useState(false);
  // "Com modelo e modo": choosing a project leads to the form instead of opening.
  const [choosing, setChoosing] = useState(false);
  const [where, setWhere] = useState<string | null>(null);
  // With "loop" on, choosing a project opens the loop's sheet there instead (D55), as in the grid's Nova sessão.
  const [looping, setLooping] = useState(false);
  // With "orquestradora" on, choosing a project opens its orchestrator instead (D66), beside the ordinary conversation.
  const [orchestrating, setOrchestrating] = useState(false);
  const groups = useMemo(() => sessionGroups(projects, query, now), [projects, query, now]);
  const total = projects.reduce((n, p) => n + p.rows.length, 0);
  const known = projects.filter((p) => p.id);
  const start = (ws: string | null, options?: StartOptions) => {
    setMenu(false);
    setWhere(null);
    onStart(ws, options);
  };
  const choose = (ws: string | null) => {
    if (orchestrating) {
      setMenu(false);
      setOrchestrating(false);
      return onOrchestrate(ws);
    }
    if (looping && ws) {
      setMenu(false);
      setLooping(false);
      return onLoop(ws);
    }
    if (!choosing) return start(ws);
    if (ws) return setWhere(ws);
    void pickFolder().then((picked) => picked && setWhere(picked));
  };
  return (
    <section className="crew-sessions" aria-label="Sessões">
      <header className="crew-pane-head">
        <h2>Sessões</h2>
        <span className="crew-new">
          <button
            type="button"
            className="crew-primary"
            aria-label="Nova sessão"
            aria-expanded={known.length > 0 ? menu : undefined}
            onClick={() => (known.length > 0 ? setMenu((m) => !m) : start(null))}
          >
            <Plus size={13} aria-hidden /> Nova {known.length > 0 && <ChevronDown size={12} aria-hidden />}
          </button>
          {menu && (
            <span className="crew-new-menu" role="menu" onKeyDown={(e) => e.key === 'Escape' && setMenu(false)}>
              {where ? (
                <StartForm workspace={where} listModels={listModels} opening={false} onOpen={(o) => start(where, o)} onBack={() => setWhere(null)} />
              ) : (
                <>
                  <span className="crew-new-title">{looping ? 'Começar loop em' : orchestrating ? 'Abrir a orquestradora de' : 'Abrir em'}</span>
                  {known.map((p) => (
                    <button key={p.id} type="button" role="menuitem" className="place" title={p.id} onClick={() => choose(p.id)}>
                      {p.label}
                    </button>
                  ))}
                  {!looping && (
                    <button type="button" role="menuitem" onClick={() => choose(null)}>
                      Outra pasta…
                    </button>
                  )}
                  <button
                    type="button"
                    role="menuitemcheckbox"
                    aria-checked={choosing}
                    className="crew-new-choose"
                    title="Escolher o modelo e o modo antes de abrir"
                    onClick={() => (setChoosing((c) => !c), setLooping(false), setOrchestrating(false))}
                  >
                    {choosing ? '✓' : '+'} com modelo e modo
                  </button>
                  <button
                    type="button"
                    aria-pressed={looping}
                    className="crew-new-choose"
                    title="Começar um loop no projeto escolhido: uma pasta de spec ou um objetivo"
                    onClick={() => (setLooping((l) => !l), setChoosing(false), setOrchestrating(false))}
                  >
                    {looping ? '✓' : '↻'} loop
                  </button>
                  {/* Hidden while the loop is chosen, as "Outra pasta…" is: that menu is a measured reference (D66). */}
                  {!looping && (
                    <button
                      type="button"
                      role="menuitemcheckbox"
                      aria-checked={orchestrating}
                      className="crew-new-choose"
                      data-orchestrate
                      title="Abrir a orquestradora do projeto escolhido: ela despacha agentes e pergunta antes de cada merge"
                      onClick={() => (setOrchestrating((o) => !o), setChoosing(false))}
                    >
                      {orchestrating ? '✓' : '◆'} orquestradora
                    </button>
                  )}
                </>
              )}
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
                g.rows.map((r) => <SessionItem key={r.id} row={r} sessions={sessions} now={now} on={r.id === focusId} depth={g.depth[r.id] ?? 0} onPick={onPick} />)}
            </div>
          );
        })}
      </div>
    </section>
  );
}
