import { useCallback, useState, type ReactNode } from 'react';
import type { SessionsState } from '../state/sessions';
import type { Row } from '../state/sidebar';
import { SheetView } from './CommandSheet';
import { MemorySection, SkillsSection } from './crew/Sections';
import type { Place } from './crew/Page';
import type { WindowActions } from './window';

export type ContextTab = 'memoria' | 'skills';

const TABS: readonly { id: ContextTab; label: string }[] = [
  { id: 'memoria', label: 'Memória' },
  { id: 'skills', label: 'Skills' },
];

/**
 * A project's memory and skills over the grid (D53): the Crew's own sections,
 * one tab each, starting on the project of the conversation they were asked
 * from. Only Crew showed them; the grid now reaches them from a window's ⋯.
 */
export function ProjectContextBody({
  tab,
  onTab,
  places,
  preferred,
  sessions,
  actions,
}: {
  tab: ContextTab;
  onTab: (tab: ContextTab) => void;
  places: readonly Place[];
  preferred: string | null;
  sessions: SessionsState;
  actions: WindowActions;
}) {
  return (
    <div className="project-context">
      <div className="project-context-tabs" role="tablist" aria-label="Contexto do projeto">
        {TABS.map((t) => (
          <button key={t.id} type="button" role="tab" aria-selected={t.id === tab} className={`project-context-tab${t.id === tab ? ' on' : ''}`} onClick={() => onTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'memoria' ? (
        <MemorySection places={places} preferred={preferred} actions={actions} />
      ) : (
        <SkillsSection places={places} preferred={preferred} sessions={sessions} actions={actions} />
      )}
    </div>
  );
}

/** The context sheet the grid has open, if any, and how a window's ⋯ opens it for its project. */
export function useProjectContext(
  places: readonly Place[],
  sessions: SessionsState,
  rows: ReadonlyMap<string, Row>,
  actions: WindowActions,
): { open: (id: string) => void; isOpen: boolean; node: ReactNode } {
  const [shown, setShown] = useState<{ workspace: string | null; tab: ContextTab } | null>(null);
  // The conversation's project: the live session's, else the list's.
  const open = useCallback(
    (id: string) => setShown({ workspace: sessions.byId[id]?.info?.workspace ?? rows.get(id)?.workspace ?? null, tab: 'memoria' }),
    [rows, sessions],
  );
  const close = useCallback(() => setShown(null), []);
  const node = shown ? (
    <SheetView title="Contexto do projeto" note="o que as sessões dele leem" onClose={close}>
      <ProjectContextBody
        tab={shown.tab}
        onTab={(tab) => setShown((s) => (s ? { ...s, tab } : s))}
        places={places}
        preferred={shown.workspace}
        sessions={sessions}
        actions={actions}
      />
    </SheetView>
  ) : null;
  return { open, isOpen: shown !== null, node };
}
