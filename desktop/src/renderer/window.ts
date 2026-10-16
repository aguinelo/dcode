// What the window is given to draw, and what it asks for. Connected, the
// answers come from the daemon through the main process (live.ts); in fixture
// mode, from a recording that sends nothing anywhere (fixture.ts).

import type { LoopRun, ModelsResponse, SpecFolder } from '../protocol/generated';
import type { Attachment, DaemonStatus, PickedAttachments, PrDraft } from '../shared/api';
import type { CommandsResponse, CompactResult, GitDiffResponse, GitPRResponse, GitResponse, MemoryResponse, SkillsResponse, UndoResult } from '../protocol/generated';
import type { History } from '../state/history';
import type { Row } from '../state/sidebar';

/** What the bottom bar says about the daemon: a live status, or that this is a recording. */
export type DaemonView = { state: 'recording'; version: string } | DaemonStatus;

/** What a new session opens on, when the person chose: a profile or model, and a mode. */
export interface StartOptions {
  model?: string;
  mode?: string;
}

/** Done, or why not — in the window's language, to be said where the person looks. */
export type Outcome<T> = { ok: true; value: T } | { ok: false; why: string };

/**
 * A PR read or opened, or why not (D60), with whose the reason is: gh not
 * ready where the daemon runs, a branch that cannot have one as things stand,
 * or a failure. The panel says each in its own tone.
 */
export type PrOutcome<T> = { ok: true; value: T } | { ok: false; why: string; kind: 'gh' | 'refused' | 'failed' };

export interface WindowActions {
  /**
   * Opens a row and answers the session to show: the row's own when it is
   * live, the new session that continues it when it ended.
   */
  open(row: Row): Promise<Outcome<string>>;
  /**
   * Sends a message: a turn for an idle session, a correction for a running
   * one (D23). Pictures (D46) and documents (D59) go with a turn only: a
   * correction carries text.
   */
  send(sessionId: string, text: string, steer: boolean, attachments?: Attachment[]): Promise<Outcome<null>>;
  /** Pictures and documents through the system's picker, read and checked by the main process (D46, D59). */
  pickAttachments(): Promise<Outcome<PickedAttachments>>;
  /** A file pasted or dropped, checked by the main process as a picked one is (D46, D59). */
  readAttachment(name: string, bytes: Uint8Array): Promise<Outcome<Attachment>>;
  stop(sessionId: string): Promise<Outcome<null>>;
  answer(sessionId: string, approvalId: string, decision: string): Promise<Outcome<null>>;
  /** Asks for a folder and opens a session there. Null when the person cancelled. */
  newSession(): Promise<Outcome<string> | null>;
  /**
   * Opens a session in a project, and answers it to show — on the model and
   * in the mode chosen, when they were (D38); else with the daemon's defaults.
   */
  newSessionIn(workspace: string, options?: StartOptions): Promise<Outcome<string>>;
  /** Asks for a folder through the system's picker; null when the person cancels. */
  pickFolder(): Promise<string | null>;
  /** Closes a live session; the daemon keeps its record. For one that was never asked anything. */
  closeSession(sessionId: string): Promise<Outcome<null>>;
  /** Keeps a live conversation's events coming, for its panel on the grid. Once each; again is harmless. */
  watch(row: Row): void;
  /** What a session in the workspace can ask for, as the daemon resolves it (D28). */
  listModels(workspace: string): Promise<Outcome<ModelsResponse>>;
  /**
   * Continues a conversation on another profile or model, and answers the
   * session to show. A live one is closed once its continuation opens (D28).
   */
  switchModel(sessionId: string, workspace: string, model: string): Promise<Outcome<string>>;
  /**
   * Switches a session's behavioural mode (D36). Not optimistic: the window
   * shows the new mode when the session's `session.mode_changed` arrives.
   */
  setMode(sessionId: string, mode: string): Promise<Outcome<null>>;
  /**
   * Names a conversation, live or ended (D43); an empty name gives the
   * automatic title back. Not optimistic: the title changes when the
   * daemon's `session.renamed` or its list says so.
   */
  rename(conversationId: string, name: string): Promise<Outcome<null>>;
  /** Runs a command the person typed (`!`), outside a turn (D40); the output arrives in the flow. */
  exec(sessionId: string, command: string): Promise<Outcome<null>>;
  /** Puts back what the last turn changed, and says what went back and what did not (D40). */
  undo(sessionId: string): Promise<Outcome<UndoResult>>;
  /** Compacts the context now, and says whether there was anything to compact (D40). */
  compact(sessionId: string): Promise<Outcome<CompactResult>>;
  /** The skills a session in the workspace has (`GET /v1/skills`, N6), or why not — a daemon without the route says so. */
  listSkills(workspace: string): Promise<Outcome<SkillsResponse>>;
  /** The slash commands the user and the project wrote, as the TUI finds them (`GET /v1/commands`, D50), or why not. */
  listCommands(workspace: string): Promise<Outcome<CommandsResponse>>;
  /** What a session in the workspace reads as memory (`GET /v1/memory`, N6), or why not. */
  listMemory(workspace: string): Promise<Outcome<MemoryResponse>>;
  /**
   * A conversation that ended, read from its record and folded like a live
   * one's events, without continuing it (D42) — or why it could not be read.
   */
  readHistory(id: string): Promise<History>;
  /** Where a conversation's folder is in git now, and what changed in it (`GET /v1/git`, D41), or why not. */
  readGit(conversationId: string): Promise<Outcome<GitResponse>>;
  /** One changed file's diff in a conversation's folder (`GET /v1/git/diff`, D41), or why not. */
  readGitDiff(conversationId: string, path: string): Promise<Outcome<GitDiffResponse>>;
  /** The PR of the branch a conversation's folder has now, with its checks (`GET /v1/git/pr`, D60), or why not. */
  readGitPR(conversationId: string): Promise<PrOutcome<GitPRResponse>>;
  /** Pushes that branch and opens the PR a person confirmed (`POST /v1/git/pr`, D60). Only the sheet calls it. */
  openGitPR(conversationId: string, draft: PrDraft): Promise<PrOutcome<GitPRResponse>>;
  /** Starts `/loop` in a project with its argument as typed (D55); a refusal says the daemon's reason. */
  startLoop(workspace: string, args: string): Promise<Outcome<LoopRun>>;
  /** Stops a run now, or after the folder it is in (D55). */
  stopLoop(loopId: string, when: string): Promise<Outcome<LoopRun>>;
  /** The spec folders of a project and what each declares, for the "Começar loop" sheet (D55). */
  listSpecs(workspace: string): Promise<Outcome<SpecFolder[]>>;
  /** "Começar o trabalho": takes the run out of awaiting_review (D56); another client may have answered first. */
  continueLoop(loopId: string): Promise<Outcome<LoopRun>>;
  /** Opens a run's proposal in the system's editor, through the main process (D56). */
  openProposal(loopId: string, file: string): Promise<Outcome<null>>;
}
