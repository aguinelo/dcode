import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron';
import type { ConversationChange, ModelsResponse, Session } from '../protocol/generated';
import {
  CHANNELS,
  type Answer,
  type Attachment,
  type DaemonStatus,
  type DcodeApi,
  type LoopsPush,
  type PickedAttachments,
  type SessionEvents,
  type StreamEnd,
  type TerminalOpened,
  type TerminalPush,
  type UserInfo,
} from '../shared/api';

// The only door between the renderer and the rest of the app. Each entry is a
// single, named request or push: the renderer never gets ipcRenderer itself,
// so it cannot send on a channel nobody here chose. Arguments cross as they
// were given; the main process checks them, and answers a bad one with a
// refusal.

/**
 * Listens on a push channel and returns what stops it. With `now`, the value
 * as it stands is asked for too, and delivered when there is one — so a
 * window that subscribes late, or reloads, starts from where things are
 * rather than from the next change. The IPC event never crosses: it carries
 * the sender, which is ipcRenderer.
 */
function subscribe<T>(channel: string, listener: (value: T) => void, now?: string): () => void {
  let listening = true;
  const handler = (_event: IpcRendererEvent, value: T): void => {
    if (listening) listener(value);
  };
  ipcRenderer.on(channel, handler);
  if (now !== undefined) {
    ipcRenderer.invoke(now).then(
      (value: T | null) => {
        if (listening && value !== null) listener(value);
      },
      (err: unknown) => console.error(`dcode: ${now} failed:`, err),
    );
  }
  return () => {
    listening = false;
    ipcRenderer.removeListener(channel, handler);
  };
}

function invoke<T>(channel: string, ...args: unknown[]): Promise<T> {
  return ipcRenderer.invoke(channel, ...args) as Promise<T>;
}

const api: DcodeApi = {
  platform: process.platform,
  user: () => invoke<UserInfo>(CHANNELS.user),

  onDaemon: (listener) => subscribe<DaemonStatus>(CHANNELS.daemon, listener, CHANNELS.daemonNow),
  onConversations: (listener) => subscribe<ConversationChange>(CHANNELS.conversations, listener, CHANNELS.conversationsNow),
  onSessionEvents: (listener) => subscribe<SessionEvents>(CHANNELS.sessionEvents, listener),
  onStreamEnd: (listener) => subscribe<StreamEnd>(CHANNELS.streamEnd, listener),

  follow: (sessionId) => invoke<Answer<null>>(CHANNELS.follow, sessionId),
  unfollow: (sessionId) => invoke<void>(CHANNELS.unfollow, sessionId),
  continueConversation: (conversation, model) =>
    model === undefined
      ? invoke<Answer<Session>>(CHANNELS.continueConversation, conversation)
      : invoke<Answer<Session>>(CHANNELS.continueConversation, conversation, model),
  closeSession: (sessionId) => invoke<Answer<null>>(CHANNELS.closeSession, sessionId),
  listModels: (workspace) => invoke<Answer<ModelsResponse>>(CHANNELS.listModels, workspace),
  listSkills: (workspace) => invoke<Answer<unknown>>(CHANNELS.listSkills, workspace),
  listCommands: (workspace) => invoke<Answer<unknown>>(CHANNELS.listCommands, workspace),
  listMemory: (workspace) => invoke<Answer<unknown>>(CHANNELS.listMemory, workspace),
  conversationEvents: (id, from) => invoke<Answer<unknown>>(CHANNELS.conversationEvents, id, from),
  readGit: (conversationId) => invoke<Answer<unknown>>(CHANNELS.readGit, conversationId),
  readGitDiff: (conversationId, path) => invoke<Answer<unknown>>(CHANNELS.readGitDiff, conversationId, path),
  pickFolder: () => invoke<string | null>(CHANNELS.pickFolder),
  createSession: (workspace, options) =>
    options === undefined
      ? invoke<Answer<Session>>(CHANNELS.createSession, workspace)
      : invoke<Answer<Session>>(CHANNELS.createSession, workspace, options),
  submitTurn: (sessionId, text, attachments) =>
    attachments && attachments.length > 0
      ? invoke<Answer<null>>(CHANNELS.submitTurn, sessionId, text, attachments)
      : invoke<Answer<null>>(CHANNELS.submitTurn, sessionId, text),
  pickAttachments: () => invoke<PickedAttachments>(CHANNELS.pickAttachments),
  readAttachment: (name, bytes) => invoke<Answer<Attachment>>(CHANNELS.readAttachment, name, bytes),
  steer: (sessionId, text) => invoke<Answer<null>>(CHANNELS.steer, sessionId, text),
  interrupt: (sessionId) => invoke<Answer<null>>(CHANNELS.interrupt, sessionId),
  resolveApproval: (sessionId, approvalId, decision) =>
    invoke<Answer<null>>(CHANNELS.resolveApproval, sessionId, approvalId, decision),
  setMode: (sessionId, mode) => invoke<Answer<null>>(CHANNELS.setMode, sessionId, mode),
  renameConversation: (id, name) => invoke<Answer<null>>(CHANNELS.renameConversation, id, name),

  copy: (text) => invoke<Answer<null>>(CHANNELS.copy, text),
  openTerminal: (sessionId, command, cols, rows) =>
    invoke<Answer<TerminalOpened>>(CHANNELS.terminalCreate, sessionId, command, cols, rows),
  // Keys and sizes go one way, as they happen: no answer to wait on per keystroke.
  writeTerminal: (id, data) => ipcRenderer.send(CHANNELS.terminalWrite, id, data),
  resizeTerminal: (id, cols, rows) => ipcRenderer.send(CHANNELS.terminalResize, id, cols, rows),
  killTerminal: (id) => ipcRenderer.send(CHANNELS.terminalKill, id),
  onTerminal: (listener) => {
    const offData = subscribe<TerminalPush>(CHANNELS.terminalData, listener);
    const offExit = subscribe<TerminalPush>(CHANNELS.terminalExit, listener);
    return () => {
      offData();
      offExit();
    };
  },
  exec: (sessionId, command) => invoke<Answer<null>>(CHANNELS.exec, sessionId, command),
  undo: (sessionId) => invoke<Answer<unknown>>(CHANNELS.undo, sessionId),
  compact: (sessionId) => invoke<Answer<unknown>>(CHANNELS.compact, sessionId),
  loops: {
    on: (listener) => subscribe<LoopsPush>(CHANNELS.loops, listener, CHANNELS.loopsNow),
    start: (workspace, args) => invoke<Answer<unknown>>(CHANNELS.startLoop, workspace, args),
    stop: (loopId, when) => invoke<Answer<unknown>>(CHANNELS.stopLoop, loopId, when),
    specs: (workspace) => invoke<Answer<unknown>>(CHANNELS.listSpecs, workspace),
    continue: (loopId) => invoke<Answer<unknown>>(CHANNELS.continueLoop, loopId),
    openProposal: (loopId, file) => invoke<Answer<null>>(CHANNELS.openProposal, loopId, file),
  },
};

contextBridge.exposeInMainWorld('dcode', api);
