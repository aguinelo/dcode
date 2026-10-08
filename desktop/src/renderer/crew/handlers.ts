import type { Look } from '../TopBar';
import type { Row } from '../../state/sidebar';

/** What the Crew look does, by the conversation in focus and the window. */
export interface CrewHandlers {
  /** A row of the sessions list was picked: it takes the middle. */
  pick: (row: Row) => void;
  draft: (id: string, text: string) => void;
  send: (id: string, text: string) => void;
  stop: (id: string) => void;
  answer: (id: string, approvalId: string, decision: string) => void;
  continueIn: (row: Row, text: string) => void;
  model: (row: Row, anchor: HTMLElement) => void;
  missing: (what: string) => void;
  /** Opens a session from the folder picker. */
  newSession: () => void;
  /** Opens a session in a known project, or from the folder picker with null; it takes the middle. */
  startIn: (workspace: string | null) => void;
  look: (look: Look) => void;
}
