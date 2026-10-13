import type { ProjectView } from '../state/sidebar';
import { nameParts } from './host';
import { ProjectList, type ProjectActions } from './ProjectList';

export function Sidebar({
  projects,
  selectedId,
  onGridIds,
  gridShown,
  gridCount,
  now,
  userName,
  actions,
  onToggleAll,
  onShowGrid,
  onNewSession,
  onSearch,
  onMissing,
}: {
  projects: ProjectView[];
  selectedId: string | null;
  /** The conversations holding a place on the grid. */
  onGridIds: ReadonlySet<string>;
  /** The grid is on the stage. */
  gridShown: boolean;
  gridCount: number;
  now: number;
  userName: string | null;
  actions: ProjectActions;
  onToggleAll: () => void;
  onShowGrid: () => void;
  onNewSession: () => void;
  onSearch: () => void;
  onMissing: (what: string) => void;
}) {
  const anyOpen = projects.some((p) => !p.collapsed);
  const user = userName ? nameParts(userName) : null;
  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        <button type="button" className={`nav-item${gridShown ? ' on' : ''}`} onClick={onShowGrid}>
          <span className="nav-icon">▦</span>
          <span className="nav-label">Grade{gridCount > 0 ? ` · ${gridCount}` : ''}</span>
          <span className="nav-key">⌘0</span>
        </button>
        <button type="button" className="nav-item" onClick={onNewSession}>
          <span className="nav-icon">+</span>
          <span className="nav-label">Nova sessão</span>
          <span className="nav-key">⌘N</span>
        </button>
        <button type="button" className="nav-item" onClick={onSearch}>
          <span className="nav-icon">⌕</span>
          <span className="nav-label">Procurar</span>
          <span className="nav-key">⌘K</span>
        </button>
      </nav>
      <div className="projects">
        <div className="projects-head">
          <span className="projects-title">Conversas</span>
          <button type="button" className="projects-toggle" onClick={onToggleAll}>
            {anyOpen ? 'recolher tudo' : 'expandir tudo'}
          </button>
          <button type="button" className="projects-add" aria-label="Adicionar projeto" onClick={() => onMissing('Adicionar projeto')}>
            +
          </button>
        </div>
        <ProjectList projects={projects} selectedId={selectedId} onGridIds={onGridIds} now={now} actions={actions} />
      </div>
      <div className="sidebar-foot">
        <span className="avatar">{user?.initials ?? '?'}</span>
        <span className="user-name">{user?.first ?? '—'}</span>
        <button type="button" className="settings" aria-label="Configurações" onClick={() => onMissing('Configurações')}>
          ⚙
        </button>
      </div>
    </aside>
  );
}
