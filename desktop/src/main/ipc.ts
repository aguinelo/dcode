import { BrowserWindow, dialog, ipcMain, type IpcMainInvokeEvent, type OpenDialogOptions } from 'electron';
import { CHANNELS } from '../shared/api';
import type { Connection } from './connection';
import * as input from './inputs';
import { currentUser } from './user';

// The window's side of the main process: one handler per named request, and
// the pushes that go out to every window. Thin on purpose — what is decided
// is decided in connection.ts and inputs.ts, which are tested without
// Electron. Every argument is checked here before it goes anywhere, and a bad
// one is answered with a refusal, never thrown at the renderer.

type Handler = (event: IpcMainInvokeEvent, ...args: unknown[]) => unknown;

/** Answers a channel for the app's own windows only. */
function handle(channel: string, answer: Handler): void {
  ipcMain.handle(channel, (event, ...args: unknown[]) => {
    if (!BrowserWindow.fromWebContents(event.sender)) throw new Error(`dcode: ${channel} asked by an unknown sender`);
    return answer(event, ...args);
  });
}

const SESSION = 'O id da sessão';

/** The system's folder picker. Called on the `dialog` object: the daemon check replaces exactly that method. */
async function pickFolder(event: IpcMainInvokeEvent): Promise<string | null> {
  const options: OpenDialogOptions = { properties: ['openDirectory', 'createDirectory'] };
  const win = BrowserWindow.fromWebContents(event.sender);
  const picked = win ? await dialog.showOpenDialog(win, options) : await dialog.showOpenDialog(options);
  if (picked.canceled) return null;
  return picked.filePaths[0] ?? null;
}

export function serveTheWindows(daemon: Connection): void {
  handle(CHANNELS.user, () => currentUser());
  handle(CHANNELS.daemonNow, () => daemon.status());
  handle(CHANNELS.conversationsNow, () => daemon.conversationsNow());
  handle(CHANNELS.pickFolder, (event) => pickFolder(event));

  handle(CHANNELS.follow, (_event, id) => {
    const session = input.id(id, SESSION);
    return session.ok ? daemon.follow(session.value) : session;
  });
  handle(CHANNELS.unfollow, (_event, id) => {
    const session = input.id(id, SESSION);
    // unfollow answers nothing, so the refusal can only be said here.
    if (session.ok) daemon.unfollow(session.value);
    else console.warn(`dcode: unfollow ignored: ${session.refusal.message}`);
  });
  const ask = daemon.requests;
  handle(CHANNELS.createSession, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.createSession(where.value) : where;
  });
  handle(CHANNELS.continueConversation, (_event, conversation, model) => {
    const which = input.conversation(conversation);
    if (!which.ok) return which;
    if (model === undefined) return ask.continueConversation(which.value.id, which.value.workspace);
    const named = input.model(model);
    return named.ok ? ask.continueConversation(which.value.id, which.value.workspace, named.value) : named;
  });
  handle(CHANNELS.closeSession, (_event, id) => {
    const session = input.id(id, SESSION);
    return session.ok ? ask.closeSession(session.value) : session;
  });
  handle(CHANNELS.listModels, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listModels(where.value) : where;
  });
  handle(CHANNELS.listSkills, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listSkills(where.value) : where;
  });
  handle(CHANNELS.listMemory, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listMemory(where.value) : where;
  });
  handle(CHANNELS.submitTurn, (_event, id, text) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const said = input.text(text);
    return said.ok ? ask.submitTurn(session.value, said.value) : said;
  });
  handle(CHANNELS.steer, (_event, id, text) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const said = input.text(text);
    return said.ok ? ask.steer(session.value, said.value) : said;
  });
  handle(CHANNELS.interrupt, (_event, id) => {
    const session = input.id(id, SESSION);
    return session.ok ? ask.interrupt(session.value) : session;
  });
  handle(CHANNELS.resolveApproval, (_event, id, approvalId, decision) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const approval = input.id(approvalId, 'O id da aprovação');
    if (!approval.ok) return approval;
    const answer = input.decision(decision);
    return answer.ok ? ask.resolveApproval(session.value, approval.value, answer.value) : answer;
  });
}

/** Sends a push to every window of the app. */
export function broadcast(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload);
  }
}
