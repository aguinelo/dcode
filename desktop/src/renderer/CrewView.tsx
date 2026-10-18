import { useState } from 'react';
import { isCrewSection, type CrewSection } from '../state/crew';
import type { History } from '../state/history';
import type { Measure } from '../state/models';
import { movedKey } from '../state/git';
import { paneView } from '../state/pane';
import type { SessionView } from '../state/session';
import type { SessionsState } from '../state/sessions';
import type { ProjectView, Row } from '../state/sidebar';
import { ContextPane } from './crew/ContextPane';
import { CrewRail } from './crew/CrewRail';
import type { CrewHandlers } from './crew/handlers';
import type { Place } from './crew/Page';
import { MemorySection, NotYetSection, SkillsSection } from './crew/Sections';
import { SettingsSection } from './crew/Settings';
import { SessionsPane } from './crew/SessionsPane';
import { ThreadPane } from './crew/ThreadPane';
import { APPS_NOT_YET, KNOWLEDGE_NOT_YET, ROUTINES_NOT_YET } from './text';
import type { ThemeChoice } from '../state/theme';
import type { Look } from './TopBar';
import { useGit } from './useGit';
import type { DaemonView, WindowActions } from './window';

export type { CrewHandlers } from './crew/handlers';

/** Which section of the rail, per window, as a convenience: the Painel when unknown (D33). */
const SECTION_KEY = 'dcode.desktop.crew-section.v1';

function loadSection(): CrewSection {
  try {
    const s = window.localStorage.getItem(SECTION_KEY);
    return isCrewSection(s) ? s : 'painel';
  } catch {
    return 'painel';
  }
}

/** A conversation of the sessions list, by id: where a child of an orchestrator opens from (RN-9). */
function rowIn(projects: readonly ProjectView[], id: string): Row | null {
  for (const p of projects) for (const r of p.rows) if (r.id === id) return r;
  return null;
}

/**
 * The second version of the screen (D32, D33): a rail of sections, and in
 * the Painel three panes — the daemon's conversations on the left, the one in
 * focus in the middle, whole, and on the right what it established. The other
 * sections show what the daemon has, or say it has nothing yet.
 */
export function CrewView({
  row,
  session,
  projects,
  places,
  sessions,
  now,
  verbTick,
  draft,
  caret,
  measure,
  waiting,
  look,
  theme,
  daemon,
  actions,
  on,
  history,
  onRetryHistory,
  onSectionError,
}: {
  /** The conversation in focus, if any. */
  row: Row | null;
  session: SessionView | undefined;
  projects: readonly ProjectView[];
  places: readonly Place[];
  sessions: SessionsState;
  now: number;
  verbTick: number;
  draft: string;
  caret?: number;
  measure: Measure | null;
  /** How many conversations wait for you, for the rail. */
  waiting: number;
  look: Look;
  theme: ThemeChoice;
  daemon: DaemonView;
  actions: WindowActions;
  on: CrewHandlers;
  /** The history of the conversation in focus when it ended, read from its record (D42). */
  history: History | undefined;
  onRetryHistory: () => void;
  /** Says a section that could not be remembered. */
  onSectionError: (text: string) => void;
}) {
  const [section, setSection] = useState<CrewSection>(loadSection);
  const pick = (s: CrewSection) => {
    setSection(s);
    try {
      window.localStorage.setItem(SECTION_KEY, s);
    } catch (err) {
      onSectionError(`Não foi possível guardar a seção (${(err as Error).message}); vale só até fechar.`);
    }
  };
  const view = row ? paneView(row, session, now) : null;
  // The folder's git, once for the thread's header and the right pane (D41).
  const moved = row ? movedKey(row, session) : null;
  const git = useGit(row?.id ?? null, moved, section === 'painel');
  const preferred = session?.info?.workspace ?? row?.workspace ?? null;
  let body: React.ReactNode;
  switch (section) {
    case 'painel':
      body = (
        <div className="crew-board">
          <SessionsPane projects={projects} sessions={sessions} focusId={row?.id ?? null} now={now} onPick={on.pick}
            onStart={on.startIn}
            onLoop={on.loop}
            listModels={actions.listModels}
            pickFolder={actions.pickFolder}
          />
          <ThreadPane
            row={row}
            view={view}
            session={session}
            now={now}
            verbTick={verbTick}
            draft={draft}
            caret={caret}
            history={history}
            onRetryHistory={onRetryHistory}
            on={on}
            git={git.state}
            measure={measure}
          />
          <ContextPane
            row={row}
            view={view}
            session={session ?? (history?.state === 'loaded' ? history.view : undefined)}
            history={session ? undefined : history}
            measure={measure}
            git={git}
            moved={moved}
            onModel={on.model}
            onMode={on.mode}
            canOpen={(id) => rowIn(projects, id) !== null}
            onOpen={(id) => {
              const r = rowIn(projects, id);
              if (r) on.pick(r);
            }}
          />
        </div>
      );
      break;
    case 'agenda':
      body = <NotYetSection title="Agenda" about="Rotinas que rodam sozinhas, num horário ou num gatilho." text={ROUTINES_NOT_YET} />;
      break;
    case 'memoria':
      body = <MemorySection places={places} preferred={preferred} actions={actions} />;
      break;
    case 'skills':
      body = <SkillsSection places={places} preferred={preferred} sessions={sessions} actions={actions} />;
      break;
    case 'apps':
      body = <NotYetSection title="Apps" about="Serviços de fora que o agente pode usar numa sessão." text={APPS_NOT_YET} />;
      break;
    case 'conhecimento':
      body = <NotYetSection title="Conhecimento" about="Documentos e fontes que o agente consulta além do código." text={KNOWLEDGE_NOT_YET} />;
      break;
    case 'config':
      body = <SettingsSection look={look} onLook={on.look} theme={theme} onTheme={on.theme} daemon={daemon} places={places} preferred={preferred} actions={actions} />;
      break;
  }
  return (
    <div className="crew-shell">
      <CrewRail section={section} waiting={waiting} onPick={pick} />
      {/* The main screen's stage (D63), the same landmark the grid's is. */}
      <main className="crew-stage">{body}</main>
    </div>
  );
}
