// What the window is given to draw, and what it asks for. Connected, the
// answers come from the daemon through the main process (live.ts); in fixture
// mode, from a recording that sends nothing anywhere (fixture.ts).

import type { DaemonStatus } from '../shared/api';
import type { Row } from '../state/sidebar';

/** What the bottom bar says about the daemon: a live status, or that this is a recording. */
export type DaemonView = { state: 'recording'; version: string } | DaemonStatus;

/** Done, or why not — in the window's language, to be said where the person looks. */
export type Outcome<T> = { ok: true; value: T } | { ok: false; why: string };

export interface WindowActions {
  /**
   * Opens a row and answers the session to show: the row's own when it is
   * live, the new session that continues it when it ended.
   */
  open(row: Row): Promise<Outcome<string>>;
  /** Sends a message: a turn for an idle session, a correction for a running one (D23). */
  send(sessionId: string, text: string, steer: boolean): Promise<Outcome<null>>;
  stop(sessionId: string): Promise<Outcome<null>>;
  answer(sessionId: string, approvalId: string, decision: string): Promise<Outcome<null>>;
  /** Asks for a folder and opens a session there. Null when the person cancelled. */
  newSession(): Promise<Outcome<string> | null>;
}
