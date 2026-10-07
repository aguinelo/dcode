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
  /** Asks for a folder through the system's picker; null when the person cancels. */
  pickFolder(): Promise<string | null>;
  /** Opens a session in a workspace, with the daemon's defaults. */
  createSession(workspace: string): Promise<Answer<Session>>;
  /** A turn, for an idle session. */
  submitTurn(sessionId: string, text: string): Promise<Answer<null>>;
  /** A correction for the turn running (D23). */
  steer(sessionId: string, text: string): Promise<Answer<null>>;
  interrupt(sessionId: string): Promise<Answer<null>>;
  resolveApproval(sessionId: string, approvalId: string, decision: string): Promise<Answer<null>>;
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
  pickFolder: 'dcode:pick-folder',
  createSession: 'dcode:create-session',
  submitTurn: 'dcode:submit-turn',
  steer: 'dcode:steer',
  interrupt: 'dcode:interrupt',
  resolveApproval: 'dcode:resolve-approval',
} as const;

/** The channel the main process answers `user()` on. */
export const USER_CHANNEL = CHANNELS.user;
