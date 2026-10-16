import { BrowserWindow, clipboard, dialog, ipcMain, shell, type IpcMainEvent, type IpcMainInvokeEvent, type OpenDialogOptions } from 'electron';
import { CHANNELS, type PickedAttachments } from '../shared/api';
import type { Connection } from './connection';
import { proposalToOpen } from './loops';
import * as attachments from './attachments';
import { DOCUMENT_EXTENSIONS } from './documents';
import { IMAGE_EXTENSIONS } from './images';
import * as input from './inputs';
import type { Terminals } from './terminals';
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

/**
 * Listens on a one-way channel, for the app's own windows only. A bad
 * argument has no one to answer, so it is said in the main process's log —
 * by its reason, never by what was typed: a terminal's keys can be a password.
 */
function listen(channel: string, act: (event: IpcMainEvent, ...args: unknown[]) => { ok: true } | { ok: false; refusal: { message: string } }): void {
  ipcMain.on(channel, (event, ...args: unknown[]) => {
    if (!BrowserWindow.fromWebContents(event.sender)) {
      console.warn(`dcode: ${channel} sent by an unknown sender, ignored`);
      return;
    }
    const done = act(event, ...args);
    if (!done.ok) console.warn(`dcode: ${channel} ignored: ${done.refusal.message}`);
  });
}

const SESSION = 'O id da sessão';
const TERMINAL = 'O id do terminal';

/** The system's folder picker. Called on the `dialog` object: the daemon check replaces exactly that method. */
async function pickFolder(event: IpcMainInvokeEvent): Promise<string | null> {
  const options: OpenDialogOptions = { properties: ['openDirectory', 'createDirectory'] };
  const win = BrowserWindow.fromWebContents(event.sender);
  const picked = win ? await dialog.showOpenDialog(win, options) : await dialog.showOpenDialog(options);
  if (picked.canceled) return null;
  return picked.filePaths[0] ?? null;
}

/**
 * The system's file picker, for pictures (D46) and documents (D59). Each
 * picked file is read and checked here; one refused is said by its reason and
 * the rest still attach.
 */
async function pickAttachments(event: IpcMainInvokeEvent): Promise<PickedAttachments> {
  const options: OpenDialogOptions = {
    properties: ['openFile', 'multiSelections'],
    filters: [
      { name: 'Imagens, PDFs e textos', extensions: [...IMAGE_EXTENSIONS, ...DOCUMENT_EXTENSIONS] },
      { name: 'Imagens', extensions: [...IMAGE_EXTENSIONS] },
      { name: 'PDFs e textos', extensions: [...DOCUMENT_EXTENSIONS] },
    ],
  };
  const win = BrowserWindow.fromWebContents(event.sender);
  const picked = win ? await dialog.showOpenDialog(win, options) : await dialog.showOpenDialog(options);
  return picked.canceled ? { attachments: [], refused: [] } : attachments.readAttachmentFiles(picked.filePaths);
}

export function serveTheWindows(daemon: Connection, terminals: Terminals): void {
  handle(CHANNELS.user, () => currentUser());
  handle(CHANNELS.daemonNow, () => daemon.status());
  handle(CHANNELS.conversationsNow, () => daemon.conversationsNow());
  handle(CHANNELS.loopsNow, () => daemon.loops.now());
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
  handle(CHANNELS.createSession, (_event, workspace, options) => {
    const where = input.workspace(workspace);
    if (!where.ok) return where;
    const how = input.sessionOptions(options);
    return how.ok ? ask.createSession(where.value, how.value) : how;
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
  handle(CHANNELS.listCommands, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listCommands(where.value) : where;
  });
  handle(CHANNELS.listMemory, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listMemory(where.value) : where;
  });
  handle(CHANNELS.conversationEvents, (_event, id, from) => {
    const which = input.recordedId(id);
    if (!which.ok) return which;
    const at = input.position(from);
    return at.ok ? ask.conversationEvents(which.value, at.value) : at;
  });
  // A conversation's git (D41): the window names the conversation, and the
  // folder is the one the daemon lists for it — never a path the window chose.
  handle(CHANNELS.readGit, (_event, id) => {
    const which = input.id(id, 'O id da conversa');
    if (!which.ok) return which;
    const where = input.gitWorkspaceOf(daemon.listed(), which.value);
    return where.ok ? ask.readGit(where.value) : where;
  });
  handle(CHANNELS.readGitDiff, (_event, id, file) => {
    const which = input.id(id, 'O id da conversa');
    if (!which.ok) return which;
    const path = input.repoPath(file);
    if (!path.ok) return path;
    const where = input.gitWorkspaceOf(daemon.listed(), which.value);
    return where.ok ? ask.readGitDiff(where.value, path.value) : where;
  });
  // A conversation's PR (D60), by the same rule: the folder is the daemon's.
  handle(CHANNELS.readGitPR, (_event, id) => {
    const which = input.id(id, 'O id da conversa');
    if (!which.ok) return which;
    const where = input.gitWorkspaceOf(daemon.listed(), which.value);
    return where.ok ? ask.readGitPR(where.value) : where;
  });
  handle(CHANNELS.createGitPR, (_event, id, draft) => {
    const which = input.id(id, 'O id da conversa');
    if (!which.ok) return which;
    const pr = input.prDraft(draft);
    if (!pr.ok) return pr;
    const where = input.gitWorkspaceOf(daemon.listed(), which.value);
    return where.ok ? ask.createGitPR(where.value, pr.value) : where;
  });
  handle(CHANNELS.submitTurn, (_event, id, text, attached) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const said = input.text(text);
    if (!said.ok) return said;
    const shown = attachments.turnAttachments(attached);
    return shown.ok ? ask.submitTurn(session.value, said.value, shown.value.images, shown.value.documents) : shown;
  });
  // Pictures (D46) and documents (D59): a picked file is read here, and pasted
  // or dropped bytes are checked here the same way. The renderer never names a path.
  handle(CHANNELS.pickAttachments, (event) => pickAttachments(event));
  handle(CHANNELS.readAttachment, (_event, name, bytes) => attachments.attachmentFromBytes(name, bytes));
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
  handle(CHANNELS.setMode, (_event, id, mode) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const named = input.mode(mode);
    return named.ok ? ask.setMode(session.value, named.value) : named;
  });
  // A conversation's name (D43), live or ended: the id in the daemon's
  // alphabet, since it becomes a path there, and the name trimmed and in size.
  handle(CHANNELS.renameConversation, (_event, id, name) => {
    const which = input.recordedId(id);
    if (!which.ok) return which;
    const named = input.conversationName(name);
    return named.ok ? daemon.rename(which.value, named.value) : named;
  });
  handle(CHANNELS.exec, (_event, id, typed) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const run = input.command(typed);
    return run.ok ? ask.exec(session.value, run.value) : run;
  });
  handle(CHANNELS.undo, (_event, id) => {
    const session = input.id(id, SESSION);
    return session.ok ? ask.undo(session.value) : session;
  });
  handle(CHANNELS.compact, (_event, id) => {
    const session = input.id(id, SESSION);
    return session.ok ? ask.compact(session.value) : session;
  });
  // A loop (D55): the project's folder, what followed /loop as it was typed,
  // and a run named in the daemon's alphabet, since it becomes a path there.
  handle(CHANNELS.startLoop, (_event, workspace, args) => {
    const where = input.workspace(workspace);
    if (!where.ok) return where;
    const typed = input.loopArgs(args);
    return typed.ok ? ask.startLoop(where.value, typed.value) : typed;
  });
  handle(CHANNELS.stopLoop, (_event, id, when) => {
    const run = input.loopId(id);
    if (!run.ok) return run;
    const at = input.stopWhen(when);
    return at.ok ? ask.stopLoop(run.value, at.value) : at;
  });
  handle(CHANNELS.continueLoop, (_event, id) => {
    const run = input.loopId(id);
    return run.ok ? ask.continueLoop(run.value) : run;
  });
  // The proposal in the editor (D56): only the file the daemon wrote for that
  // run, as it listed it. The renderer never names a path that is opened.
  handle(CHANNELS.openProposal, async (_event, id, file) => {
    const run = input.loopId(id);
    if (!run.ok) return run;
    const which = proposalToOpen(daemon.loops.listed(), run.value, file);
    if (!which.ok) return which;
    const failed = await shell.openPath(which.value);
    return failed ? { ok: false, refusal: { code: 'open_failed', message: `${which.value} não abriu: ${failed}` } } : { ok: true, value: null };
  });
  handle(CHANNELS.listSpecs, (_event, workspace) => {
    const where = input.workspace(workspace);
    return where.ok ? ask.listSpecs(where.value) : where;
  });
  handle(CHANNELS.resolveApproval, (_event, id, approvalId, decision) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const approval = input.id(approvalId, 'O id da aprovação');
    if (!approval.ok) return approval;
    const answer = input.decision(decision);
    return answer.ok ? ask.resolveApproval(session.value, approval.value, answer.value) : answer;
  });

  handle(CHANNELS.copy, (_event, text) => {
    const what = input.clip(text);
    if (!what.ok) return what;
    clipboard.writeText(what.value);
    return { ok: true, value: null };
  });
  // The terminal (D37): the window names the session, and the folder is the
  // one the daemon lists for it — never a path the window chose.
  handle(CHANNELS.terminalCreate, (event, id, command, cols, rows) => {
    const session = input.id(id, SESSION);
    if (!session.ok) return session;
    const line = input.command(command);
    if (!line.ok) return line;
    const cells = input.size(cols, rows);
    if (!cells.ok) return cells;
    const where = input.workspaceOf(daemon.listed(), session.value);
    if (!where.ok) return where;
    return terminals.open(event.sender.id, where.value, line.value, cells.value.cols, cells.value.rows);
  });
  listen(CHANNELS.terminalWrite, (event, id, data) => {
    const which = input.id(id, TERMINAL);
    if (!which.ok) return which;
    const typed = input.keys(data);
    return typed.ok ? terminals.write(event.sender.id, which.value, typed.value) : typed;
  });
  listen(CHANNELS.terminalResize, (event, id, cols, rows) => {
    const which = input.id(id, TERMINAL);
    if (!which.ok) return which;
    const cells = input.size(cols, rows);
    return cells.ok ? terminals.resize(event.sender.id, which.value, cells.value.cols, cells.value.rows) : cells;
  });
  listen(CHANNELS.terminalKill, (event, id) => {
    const which = input.id(id, TERMINAL);
    return which.ok ? terminals.kill(event.sender.id, which.value) : which;
  });
}

/** Sends a push to every window of the app. */
export function broadcast(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload);
  }
}
