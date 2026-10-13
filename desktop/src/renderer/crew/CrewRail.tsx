import { BookOpen, Brain, CalendarClock, LayoutGrid, PanelsTopLeft, Settings, Sparkles, type LucideIcon } from 'lucide-react';
import type { CrewSection } from '../../state/crew';

interface Item {
  id: CrewSection;
  label: string;
  icon: LucideIcon;
}

const MAIN: readonly Item[] = [
  { id: 'painel', label: 'Painel', icon: PanelsTopLeft },
  { id: 'agenda', label: 'Agenda', icon: CalendarClock },
  { id: 'memoria', label: 'Memória', icon: Brain },
  { id: 'skills', label: 'Skills', icon: Sparkles },
  { id: 'apps', label: 'Apps', icon: LayoutGrid },
  { id: 'conhecimento', label: 'Conhecimento', icon: BookOpen },
];

const SETTINGS: Item = { id: 'config', label: 'Configurações', icon: Settings };

function RailButton({ item, on, onPick, badge }: { item: Item; on: boolean; onPick: (s: CrewSection) => void; badge?: number }) {
  const Icon = item.icon;
  return (
    <button
      type="button"
      className={`rail-item${on ? ' on' : ''}`}
      aria-label={item.label}
      aria-current={on ? 'page' : undefined}
      title={item.label}
      data-crew-section={item.id}
      onClick={() => onPick(item.id)}
    >
      <Icon size={19} strokeWidth={1.6} aria-hidden />
      {badge !== undefined && badge > 0 && <span className="rail-badge">{badge}</span>}
    </button>
  );
}

/**
 * The Crew look's narrow rail: one icon per section, the settings at the
 * foot. The Painel carries how many conversations wait for you — the one
 * number worth seeing from any section.
 */
export function CrewRail({ section, waiting, onPick }: { section: CrewSection; waiting: number; onPick: (s: CrewSection) => void }) {
  return (
    <nav className="crew-rail" aria-label="Seções">
      <span className="rail-mark" aria-hidden>
        d
      </span>
      <div className="rail-items">
        {MAIN.map((item) => (
          <RailButton key={item.id} item={item} on={item.id === section} onPick={onPick} badge={item.id === 'painel' ? waiting : undefined} />
        ))}
      </div>
      <RailButton item={SETTINGS} on={section === 'config'} onPick={onPick} />
    </nav>
  );
}
