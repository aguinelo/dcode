import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron';
import type { ConversationChange, ModelsResponse, Session } from '../protocol/generated';
import {
  CHANNELS,
  type Answer,
  type DaemonStatus,
  type DcodeApi,
  type SessionEvents,
  type StreamEnd,
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
  listMemory: (workspace) => invoke<Answer<unknown>>(CHANNELS.listMemory, workspace),
  pickFolder: () => invoke<string | null>(CHANNELS.pickFolder),
  createSession: (workspace) => invoke<Answer<Session>>(CHANNELS.createSession, workspace),
  submitTurn: (sessionId, text) => invoke<Answer<null>>(CHANNELS.submitTurn, sessionId, text),
  steer: (sessionId, text) => invoke<Answer<null>>(CHANNELS.steer, sessionId, text),
  interrupt: (sessionId) => invoke<Answer<null>>(CHANNELS.interrupt, sessionId),
  resolveApproval: (sessionId, approvalId, decision) =>
    invoke<Answer<null>>(CHANNELS.resolveApproval, sessionId, approvalId, decision),
};

contextBridge.exposeInMainWorld('dcode', api);
