import type { ProjectView } from '../state/sidebar';
import { nameParts } from './host';
import { ProjectList, type ProjectActions } from './ProjectList';

const NEW_SESSION = 'Nova sessão';

const NAV = [
  { icon: '+', label: NEW_SESSION, key: '⌘N' },
  { icon: '⌕', label: 'Procurar', key: '⌘K' },
  { icon: '◷', label: 'Rotinas', key: '' },
];

export function Sidebar({
  drawsWindowControls,
  projects,
  selectedId,
  now,
  userName,
  canGoBack,
  canGoForward,
  actions,
  onBack,
  onForward,
  onToggleAll,
  onNewSession,
  onMissing,
}: {
  drawsWindowControls: boolean;
  projects: ProjectView[];
  selectedId: string | null;
  now: number;
  userName: string | null;
  canGoBack: boolean;
  canGoForward: boolean;
  actions: ProjectActions;
  onBack: () => void;
  onForward: () => void;
  onToggleAll: () => void;
  onNewSession: () => void;
  onMissing: (what: string) => void;
}) {
  const anyOpen = projects.some((p) => !p.collapsed);
  const user = userName ? nameParts(userName) : null;
  return (
    <aside className="sidebar">
      <div className="sidebar-top drag-region">
        {drawsWindowControls && (
          // Placeholders where macOS draws the window controls. Only in a plain
          // browser, where there are none: in Electron the OS draws its own.
          <>
            <span className="light light-close" />
            <span className="light light-min" />
            <span className="light light-max" />
          </>
        )}
        <span className="spacer" />
        <button type="button" className="sidebar-toggle no-drag" aria-label="Recolher a lateral" onClick={() => onMissing('Recolher a lateral')} />
        <button type="button" className="nav-arrow back no-drag" aria-label="Voltar" disabled={!canGoBack} onClick={onBack}>
          ←
        </button>
        <button type="button" className="nav-arrow no-drag" aria-label="Avançar" disabled={!canGoForward} onClick={onForward}>
          →
        </button>
      </div>
      <nav className="sidebar-nav">
        {NAV.map((n) => (
          <button key={n.label} type="button" className="nav-item" onClick={() => (n.label === NEW_SESSION ? onNewSession() : onMissing(n.label))}>
            <span className="nav-icon">{n.icon}</span>
            <span className="nav-label">{n.label}</span>
            <span className="nav-key">{n.key}</span>
          </button>
        ))}
      </nav>
      <div className="projects">
        <div className="projects-head">
          <span className="projects-title">Projetos</span>
          <button type="button" className="projects-toggle" onClick={onToggleAll}>
            {anyOpen ? 'recolher tudo' : 'expandir tudo'}
          </button>
          <button type="button" className="projects-add" aria-label="Adicionar projeto" onClick={() => onMissing('Adicionar projeto')}>
            +
          </button>
        </div>
        <ProjectList projects={projects} selectedId={selectedId} now={now} actions={actions} />
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
