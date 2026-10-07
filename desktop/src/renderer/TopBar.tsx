import { DaemonBar } from './DaemonBar';
import type { Host } from './host';
import type { DaemonView } from './window';

/** The two versions of the screen: the grid of windows, and Crew's three panels (D32). */
export type Look = 'grade' | 'crew';

/**
 * The bar across the top: where the window stands with the daemon, and how
 * much is asking for the person — the two things worth reading from anywhere.
 */
export function TopBar({
  host,
  daemon,
  counts,
  overflow,
  onGrid,
  canGoBack,
  canGoForward,
  onBack,
  onForward,
  look,
  onLook,
}: {
  host: Host;
  daemon: DaemonView;
  counts: { running: number; blocked: number };
  /** Conversations that want a place on the grid and found none. */
  overflow: number;
  /** The grid is on the stage: the shortcut to it is not offered. */
  onGrid: boolean;
  canGoBack: boolean;
  canGoForward: boolean;
  onBack: () => void;
  onForward: () => void;
  look: Look;
  onLook: (look: Look) => void;
}) {
  // macOS draws its window controls over the bar's left end (main.ts).
  const native = host.kind === 'electron' && host.platform === 'darwin';
  return (
    <header className={`topbar drag-region${native ? ' native-controls' : ''}`}>
      {host.drawsWindowControls && (
        // Placeholders where macOS draws the controls: only in a plain browser.
        <span className="lights">
          <span className="light" />
          <span className="light" />
          <span className="light" />
        </span>
      )}
      <button type="button" className="nav-arrow" aria-label="Voltar" disabled={!canGoBack} onClick={onBack}>
        ←
      </button>
      <button type="button" className="nav-arrow" aria-label="Avançar" disabled={!canGoForward} onClick={onForward}>
        →
      </button>
      <span className="brand">dcode</span>
      <DaemonBar daemon={daemon} />
      {counts.running > 0 && <span className="topbar-count">{counts.running} rodando</span>}
      {counts.blocked > 0 && <span className="topbar-count waiting">{counts.blocked} esperando você</span>}
      {overflow > 0 && <span className="topbar-count">+{overflow} sem lugar na grade</span>}
      <span className="spacer" />
      <span className="look-switch" role="group" aria-label="Versão da tela">
        {(['grade', 'crew'] as const).map((l) => (
          <button key={l} type="button" className={l === look ? 'on' : ''} aria-pressed={l === look} onClick={() => onLook(l)}>
            {l === 'grade' ? 'Grade' : 'Crew'}
          </button>
        ))}
      </span>
      <span className="topbar-hints">{onGrid ? '⌘N nova · ⌘↵ maximizar · ⌘K procurar' : 'esc grade · ⌘K procurar'}</span>
    </header>
  );
}
