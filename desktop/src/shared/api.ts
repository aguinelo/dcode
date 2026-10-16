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

/**
 * A picture attached to a turn (D46), read and checked by the main process:
 * its type from its bytes, never from its name. The renderer holds it to show
 * and to send, and never a path.
 */
export interface ImageAttachment {
  /** The file's name, or what the clipboard called it — only to be shown. */
  name: string;
  /** image/png, image/jpeg, image/gif or image/webp. */
  mediaType: string;
  /** The bytes, in base64, as `TurnImage.data` carries them. */
  data: string;
  /** How many bytes the picture has. */
  bytes: number;
}

/**
 * A document attached to a turn (D59): a PDF, known by its bytes, or a text
 * file, UTF-8 without NUL. Read and checked by the main process like a
 * picture, and held by the renderer the same way: to show and to send.
 */
export interface DocumentAttachment {
  /** The file's name, never a path: the daemon shows it on the message and to the model. */
  name: string;
  /** application/pdf, text/plain, text/markdown or text/csv. */
  mediaType: string;
  /** The bytes, in base64, as `TurnDocument.data` carries them. */
  data: string;
  /** How many bytes the file has. */
  bytes: number;
}

/** What a field holds for its next message: pictures and documents, told apart by their type. */
export type Attachment = ImageAttachment | DocumentAttachment;

/**
 * Whether an attachment is a picture. The main process typed it from its
 * bytes. A plain boolean rather than a type guard: the two kinds have the same
 * shape, so the type cannot tell them apart — only the media type does.
 */
export function isImage(a: Attachment): boolean {
  return a.mediaType.startsWith('image/');
}

/** What the picker gave: the files read, and why each one refused was refused. */
export interface PickedAttachments {
  attachments: Attachment[];
  refused: string[];
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

/**
 * A push of the stream of loops (D55): a frame as it came, decoded by the
 * renderer, or why the daemon gives none — one older than `/loop` in the
 * daemon, said once and never taken for a list with nothing in it.
 */
export type LoopsPush = { kind: 'frame'; frame: unknown } | { kind: 'refused'; reason: string };

/**
 * The `/loop` runs the daemon conducts (D55). Answers come as they came, and
 * the renderer reads them at its boundary.
 */
export interface LoopApi {
  /** Every run first (a `snapshot` frame), then each one that moves. Returns what stops the listening. */
  on(listener: (push: LoopsPush) => void): () => void;
  /** Starts `/loop` in a project, with what followed `/loop` as it was typed (`POST /loops`). */
  start(workspace: string, args: string): Promise<Answer<unknown>>;
  /** Stops a run now, or after the folder it is in (`POST /loops/{id}/stop`). */
  stop(loopId: string, when: string): Promise<Answer<unknown>>;
  /** The spec folders of a project and what each declares, without running anything (`GET /specs?measure=false`). */
  specs(workspace: string): Promise<Answer<unknown>>;
  /** Takes a run out of awaiting_review and into the work (`POST /loops/{id}/continue`, D56). */
  continue(loopId: string): Promise<Answer<unknown>>;
  /**
   * Opens a run's proposal in the system's editor (D56): only a file the
   * daemon wrote for that run, checked by the main process.
   */
  openProposal(loopId: string, file: string): Promise<Answer<null>>;
}

/** A new PR as a person saw it and confirmed it (D60), every field as shown. */
export interface PrDraft {
  title: string;
  body: string;
  /** Empty: the repository's default branch, as the daemon reads it. */
  base: string;
  draft: boolean;
}

/** A conversation's branch on GitHub, through the daemon's gh (D60). Named by the conversation, never by a folder. */
export interface PrApi {
  /** The PR of the branch the conversation's folder has now, with its checks (`GET /git/pr`), as it came. */
  read(conversationId: string): Promise<Answer<unknown>>;
  /**
   * Pushes that branch and opens the PR a person confirmed (`POST /git/pr`),
   * as it came. Only ever sent from the confirmation sheet; it never merges.
   */
  open(conversationId: string, draft: PrDraft): Promise<Answer<unknown>>;
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
  /** The skills a session in the workspace has (`GET /skills`), as it came. Decoded by the renderer. */
  listSkills(workspace: string): Promise<Answer<unknown>>;
  /** The slash commands the user and the project wrote (`GET /commands`), as it came. Decoded by the renderer. */
  listCommands(workspace: string): Promise<Answer<unknown>>;
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
  /** A turn, for an idle session, with the pictures (D46) and documents (D59) attached to it. */
  submitTurn(sessionId: string, text: string, attachments?: Attachment[]): Promise<Answer<null>>;
  /** Asks for pictures and documents through the system's picker, read by the main process (D46, D59). Nothing picked is an empty list. */
  pickAttachments(): Promise<PickedAttachments>;
  /** A file pasted or dropped, as bytes: the main process checks it as it checks a picked one (D46, D59). */
  readAttachment(name: string, bytes: Uint8Array): Promise<Answer<Attachment>>;
  /** A correction for the turn running (D23). */
  steer(sessionId: string, text: string): Promise<Answer<null>>;
  interrupt(sessionId: string): Promise<Answer<null>>;
  resolveApproval(sessionId: string, approvalId: string, decision: string): Promise<Answer<null>>;
  /** Switches a session's behavioural mode: plan, assist or auto. The session's own event confirms it. */
  setMode(sessionId: string, mode: string): Promise<Answer<null>>;
  /**
   * Names a conversation, live or ended (`POST /sessions/{id}/name`, D43); an
   * empty name gives the automatic title back. The title changes when the
   * list does, never on the answer alone.
   */
  renameConversation(id: string, name: string): Promise<Answer<null>>;

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
  /** The `/loop` runs: their stream, and starting and stopping one (D55). */
  readonly loops: LoopApi;
  /** A conversation's PR: reading it, and opening one a person confirmed (D60). */
  readonly pr: PrApi;
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
  listCommands: 'dcode:list-commands',
  listMemory: 'dcode:list-memory',
  conversationEvents: 'dcode:conversation-events',
  readGit: 'dcode:read-git',
  readGitDiff: 'dcode:read-git-diff',
  readGitPR: 'dcode:read-git-pr',
  createGitPR: 'dcode:create-git-pr',
  pickFolder: 'dcode:pick-folder',
  createSession: 'dcode:create-session',
  submitTurn: 'dcode:submit-turn',
  pickAttachments: 'dcode:pick-attachments',
  readAttachment: 'dcode:read-attachment',
  steer: 'dcode:steer',
  interrupt: 'dcode:interrupt',
  resolveApproval: 'dcode:resolve-approval',
  setMode: 'dcode:set-mode',
  renameConversation: 'dcode:rename-conversation',
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
  loops: 'dcode:loops',
  loopsNow: 'dcode:loops-now',
  startLoop: 'dcode:start-loop',
  stopLoop: 'dcode:stop-loop',
  listSpecs: 'dcode:list-specs',
  continueLoop: 'dcode:continue-loop',
  openProposal: 'dcode:open-proposal',
} as const;

/** The channel the main process answers `user()` on. */
export const USER_CHANNEL = CHANNELS.user;
