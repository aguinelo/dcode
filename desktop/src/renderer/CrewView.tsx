import { useState } from 'react';
import { isCrewSection, type CrewSection } from '../state/crew';
import type { Measure } from '../state/models';
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
import type { Look } from './TopBar';
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
  daemon,
  actions,
  on,
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
  daemon: DaemonView;
  actions: WindowActions;
  on: CrewHandlers;
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
  const preferred = session?.info?.workspace ?? row?.workspace ?? null;
  let body: React.ReactNode;
  switch (section) {
    case 'painel':
      body = (
        <div className="crew-board">
          <SessionsPane projects={projects} sessions={sessions} focusId={row?.id ?? null} now={now} onPick={on.pick} onStart={on.startIn} />
          <ThreadPane row={row} view={view} session={session} now={now} verbTick={verbTick} draft={draft} caret={caret} on={on} />
          <ContextPane row={row} view={view} session={session} measure={measure} onModel={on.model} />
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
      body = <SettingsSection look={look} onLook={on.look} daemon={daemon} places={places} preferred={preferred} actions={actions} />;
      break;
  }
  return (
    <div className="crew-shell">
      <CrewRail section={section} waiting={waiting} onPick={pick} />
      <div className="crew-stage">{body}</div>
    </div>
  );
}
