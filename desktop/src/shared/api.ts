// The whole surface the preload hands the renderer. Narrow on purpose: the
// renderer never sees Node, the filesystem or the daemon's socket, only these.
//
// Each request is one named channel, and so is each push from the main
// process: nothing generic crosses, so the renderer cannot send on a channel
// nobody here chose. The main process is the daemon's client (D2, D19); the
// renderer asks it for one thing at a time and is told what changed.

import type { Conversation, ConversationChange, ModelsResponse, Session } from '../protocol/generated';

export interface UserInfo {
  /** The person's name as the system knows it, or their login when it has none. */
  name: string;
}

/**
 * Where the window stands with the daemon — what the bottom bar says.
 *
 * `lost` is a daemon that answered and stopped; `failed` is one that never
 * answered, because it could not be found, started or reached. Both carry the
 * reason in the window's language, and neither is ever shown as connected.
 */
export type DaemonStatus =
  | { state: 'connecting' }
  | {
      state: 'connected';
      /** What `GET /version` said, as the daemon said it. */
      version: string;
      socket: string;
      /** The app started this daemon, and stops it when it quits (D19). */
      started: boolean;
    }
  | { state: 'lost'; reason: string }
  | { state: 'failed'; reason: string };

/** A request the daemon refused, or one that never reached it. Said, never dropped. */
export interface Refusal {
  /** The daemon's error code (`Error.code` on the wire), or `unreachable` when no daemon took the request. */
  code: string;
  /** The daemon's message as it sent it, or why the request did not reach one. */
  message: string;
}

export const UNREACHABLE = 'unreachable';

export type Answer<T> = { ok: true; value: T } | { ok: false; refusal: Refusal };

/** Events of one followed session, in the order its stream carried them. Decoded by the renderer. */
export interface SessionEvents {
  sessionId: string;
  events: unknown[];
}

/**
 * A followed session's stream ended and will not resume: the daemon no longer
 * has the session, or no longer has the events it would resume from.
 */
export interface StreamEnd {
  sessionId: string;
  reason: string;
}

/**
 * What a new session opens on, when the person chose (`CreateSessionRequest`).
 * Each is sent only when given: what is not chosen stays the daemon's default.
 */
export interface NewSessionOptions {
  /** A profile or a model, as the daemon resolves it. */
  model?: string;
  /** read-only, workspace-write or full-access. */
  sandboxMode?: string;
}

/** A terminal the main process opened for this window (D37). */
export interface TerminalOpened {
  id: string;
  /** The session's workspace, where the shell started. */
  cwd: string;
  /** The shell's path. */
  shell: string;
}

/** What a terminal sends the window that opened it: its output as it comes, then how its shell ended. */
export type TerminalPush = { kind: 'data'; id: string; data: string } | { kind: 'exit'; id: string; code: number };

export interface DcodeApi {
  /** process.platform of the main process: decides where the native window controls are. */
  readonly platform: string;
  user(): Promise<UserInfo>;

  /** The daemon's status now, then each change. Returns what stops the listening. */
  onDaemon(listener: (status: DaemonStatus) => void): () => void;
  /**
   * The list of conversations: the whole of it first (kind `snapshot`), then
   * what changes. A reconnection opens with a new snapshot, applied like the
   * first. Returns what stops the listening.
   */
  onConversations(listener: (change: ConversationChange) => void): () => void;
  /** The events of every session the window follows. Returns what stops the listening. */
  onSessionEvents(listener: (batch: SessionEvents) => void): () => void;
  /** A followed session's stream that ended for good. Returns what stops the listening. */
  onStreamEnd(listener: (end: StreamEnd) => void): () => void;

  /**
   * Follows a live session from its first event. Following one already
   * followed sends its events again from the first — the fold ignores a seq it
   * has seen, so a reloaded window catches up the same way.
   */
  follow(sessionId: string): Promise<Answer<null>>;
  unfollow(sessionId: string): Promise<void>;
  /**
   * Continues a conversation in a new session (`CreateSessionRequest.resume`).
   * With `model` — a profile or a model, as the daemon resolves it — the
   * continuation runs on it: that is how a conversation changes model (D28).
   */
  continueConversation(conversation: Pick<Conversation, 'id' | 'workspace'>, model?: string): Promise<Answer<Session>>;
  /** Closes a live session. The daemon keeps its record, and the list shows it ended. */
  closeSession(sessionId: string): Promise<Answer<null>>;
  /** What a session in the workspace can ask for (`GET /models`), as it came. Decoded by the renderer. */
  listModels(workspace: string): Promise<Answer<ModelsResponse>>;
  /** The skills a session in the workspace has (`GET /skills`), as it came. Decoded by the renderer. */
  listSkills(workspace: string): Promise<Answer<unknown>>;
  /** What a session in the workspace reads as memory (`GET /memory`), as it came. Decoded by the renderer. */
  listMemory(workspace: string): Promise<Answer<unknown>>;
  /**
   * One page of a recorded conversation's events (`GET /conversations/{id}/events`),
   * from a position, as it came. Read-only: nothing is loaded or continued (D42).
   */
  conversationEvents(id: string, from: number): Promise<Answer<unknown>>;
  /**
   * Where a conversation's folder is in git now, and what changed in it
   * (`GET /git`), as it came (D41). Named by the conversation: the folder is
   * the one the daemon lists for it.
   */
  readGit(conversationId: string): Promise<Answer<unknown>>;
  /** One changed file's diff in a conversation's folder (`GET /git/diff`), as it came (D41). */
  readGitDiff(conversationId: string, path: string): Promise<Answer<unknown>>;
  /** Asks for a folder through the system's picker; null when the person cancels. */
  pickFolder(): Promise<string | null>;
  /** Opens a session in a workspace, on what was chosen, else with the daemon's defaults. */
  createSession(workspace: string, options?: NewSessionOptions): Promise<Answer<Session>>;
  /** A turn, for an idle session. */
  submitTurn(sessionId: string, text: string): Promise<Answer<null>>;
  /** A correction for the turn running (D23). */
  steer(sessionId: string, text: string): Promise<Answer<null>>;
  interrupt(sessionId: string): Promise<Answer<null>>;
  resolveApproval(sessionId: string, approvalId: string, decision: string): Promise<Answer<null>>;
  /** Switches a session's behavioural mode: plan, assist or auto. The session's own event confirms it. */
  setMode(sessionId: string, mode: string): Promise<Answer<null>>;

  /** Puts text on the system's clipboard. */
  copy(text: string): Promise<Answer<null>>;
  /**
   * Opens a shell in a real PTY, in the session's workspace as the daemon
   * lists it, with `command` typed on its line and not run (D37).
   */
  openTerminal(sessionId: string, command: string, cols: number, rows: number): Promise<Answer<TerminalOpened>>;
  /** Keys the person pressed, as they were pressed. Only to a terminal this window opened. */
  writeTerminal(id: string, data: string): void;
  resizeTerminal(id: string, cols: number, rows: number): void;
  /** Ends the shell and what runs in it. */
  killTerminal(id: string): void;
  /** Output and ends of this window's terminals. Returns what stops the listening. */
  onTerminal(listener: (push: TerminalPush) => void): () => void;
  /** Runs a command the person typed, outside a turn; its output comes as a typed tool call. */
  exec(sessionId: string, command: string): Promise<Answer<null>>;
  /** Puts back what the last turn changed, as it came. Decoded by the renderer. */
  undo(sessionId: string): Promise<Answer<unknown>>;
  /** Compacts the context now, as it came. Decoded by the renderer. */
  compact(sessionId: string): Promise<Answer<unknown>>;
}

/** The channels, one per request and one per push. */
export const CHANNELS = {
  user: 'dcode:user',
  daemon: 'dcode:daemon',
  daemonNow: 'dcode:daemon-now',
  conversations: 'dcode:conversations',
  conversationsNow: 'dcode:conversations-now',
  sessionEvents: 'dcode:session-events',
  streamEnd: 'dcode:stream-end',
  follow: 'dcode:follow',
  unfollow: 'dcode:unfollow',
  continueConversation: 'dcode:continue-conversation',
  closeSession: 'dcode:close-session',
  listModels: 'dcode:list-models',
  listSkills: 'dcode:list-skills',
  listMemory: 'dcode:list-memory',
  conversationEvents: 'dcode:conversation-events',
  readGit: 'dcode:read-git',
  readGitDiff: 'dcode:read-git-diff',
  pickFolder: 'dcode:pick-folder',
  createSession: 'dcode:create-session',
  submitTurn: 'dcode:submit-turn',
  steer: 'dcode:steer',
  interrupt: 'dcode:interrupt',
  resolveApproval: 'dcode:resolve-approval',
  setMode: 'dcode:set-mode',
  copy: 'dcode:copy',
  terminalCreate: 'dcode:terminal-create',
  terminalWrite: 'dcode:terminal-write',
  terminalResize: 'dcode:terminal-resize',
  terminalKill: 'dcode:terminal-kill',
  terminalData: 'dcode:terminal-data',
  terminalExit: 'dcode:terminal-exit',
  exec: 'dcode:exec',
  undo: 'dcode:undo',
  compact: 'dcode:compact',
} as const;

/** The channel the main process answers `user()` on. */
export const USER_CHANNEL = CHANNELS.user;
