// The window connected: the daemon's status, its list of conversations and the
// events of the sessions this window opened, kept as one snapshot the window
// draws. Everything arrives from the main process through the preload's API;
// nothing here touches a socket (D2).

import { decodeChange } from '../protocol/conversations';
import { decodeEvent } from '../protocol/events';
import { decodeModels } from '../protocol/models';
import { CodeGhUnavailable, CodeNoActiveTurn, CodePRRefused, CodeTurnAlreadyActive, type CompactResult, type GitPRResponse, type UndoResult } from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { Answer, Attachment, DaemonStatus, DcodeApi, NewSessionOptions } from '../shared/api';
import { STEER_TAKES_TEXT } from '../state/attachments';
import { modeToSwitch, sandboxOf } from '../state/mode';
import { applyListChange, emptyConversations, liveContinuationOf, type ConversationsState } from '../state/conversations';
import { applyDecoded, emptySessions, type SessionsState } from '../state/sessions';
import { readHistory } from '../state/history';
import { rowOfConversation, type Row } from '../state/sidebar';
import { decodeCommands } from '../protocol/commands';
import { decodeMemory, decodeSkills } from '../protocol/crew';
import { decodeGit, decodeGitDiff, decodeGitPR } from '../protocol/git';
import { decodeLoopChange, decodeLoopRun, decodeSpecs } from '../protocol/loops';
import { applyLoopChange, emptyLoops, loopsUnavailable, startRefusal, type LoopsState } from '../state/loop';
import { continueRefusal } from '../state/review';
import type { LoopsPush } from '../shared/api';
import { notExposed } from './text';
import type { Outcome, PrOutcome, StartOptions, WindowActions } from './window';

export interface LiveSnapshot {
  daemon: DaemonStatus;
  conversations: ConversationsState;
  /** The sidebar's rows, derived from the list. */
  rows: Row[];
  /** The sessions this window follows, folded from their events. */
  sessions: SessionsState;
  /** The `/loop` runs the daemon conducts (D55). */
  loops: LoopsState;
}

/** How long the list has to show a conversation ended before its stream's end is called a failure. */
const ENDED_GRACE_MS = 1500;

/** A refusal said as the window says it: what did not happen, then the daemon's reason. */
function outcome<T>(answer: Answer<T>, what: string): Outcome<T> {
  return answer.ok ? { ok: true, value: answer.value } : { ok: false, why: `${what}: ${answer.refusal.message}` };
}

/** An answer the daemon gave, read at the boundary: a shape it does not have is said, never guessed from. */
function decoded<T>(got: Outcome<unknown>, decode: (raw: unknown) => T | string): Outcome<T> {
  if (!got.ok) return got;
  const read = decode(got.value);
  return typeof read === 'string' ? { ok: false, why: `Resposta ilegível do daemon: ${read}` } : { ok: true, value: read };
}

function strings(v: unknown): v is string[] {
  return Array.isArray(v) && v.every((x) => typeof x === 'string');
}

function decodeUndo(raw: unknown): UndoResult | string {
  if (!isRecord(raw)) return 'o desfazer não trouxe um objeto';
  // A Go nil slice arrives as null: nothing restored, nothing refused.
  const restored = raw.restored ?? [];
  const refused = raw.refused ?? [];
  return strings(restored) && strings(refused) ? { restored, refused } : 'o desfazer não trouxe as listas restored e refused';
}

function decodeCompact(raw: unknown): CompactResult | string {
  return isRecord(raw) && typeof raw.compacted === 'boolean' ? { compacted: raw.compacted } : 'o resumo não disse se compactou';
}

/** The main process's code for an answer it could not read as the daemon's (main/wire.ts). */
const UNEXPECTED = 'unexpected_answer';

/**
 * A listing read at the boundary. A 404 is a daemon older than the route,
 * which is said as such with its own words; any other refusal is a failure,
 * and an answer that does not match the shape is said, never guessed from.
 */
function read<T>(
  answer: Answer<unknown>,
  decode: (raw: unknown) => { ok: true; value: T } | { ok: false; reason: string },
  what: string,
  failed: string,
): Outcome<T> {
  if (!answer.ok) {
    const older = answer.refusal.code === UNEXPECTED && / 404 /.test(answer.refusal.message);
    return { ok: false, why: older ? notExposed(what, answer.refusal.message) : `${failed}: ${answer.refusal.message}` };
  }
  const got = decode(answer.value);
  return got.ok ? got : { ok: false, why: `Resposta ilegível do daemon: ${got.reason}` };
}

/**
 * A PR answer (D60): the daemon's refusal kept with whose it is — gh not ready,
 * or a branch that cannot have a PR — so the panel says each in its tone.
 */
export function prOutcome(answer: Answer<unknown>, what: string, failed: string): PrOutcome<GitPRResponse> {
  if (!answer.ok && answer.refusal.code === CodeGhUnavailable) return { ok: false, why: answer.refusal.message, kind: 'gh' };
  if (!answer.ok && answer.refusal.code === CodePRRefused) return { ok: false, why: answer.refusal.message, kind: 'refused' };
  const got = read(answer, decodeGitPR, what, failed);
  return got.ok ? got : { ...got, kind: 'failed' };
}

export class LiveStore {
  private snap: LiveSnapshot = {
    daemon: { state: 'connecting' },
    conversations: emptyConversations,
    rows: [],
    sessions: emptySessions,
    loops: emptyLoops,
  };
  private readonly listeners = new Set<() => void>();
  private readonly noticed = new Set<(text: string) => void>();
  /** Continuations on their way, so a second open of the same ended conversation waits for the first. */
  private readonly continuing = new Map<string, Promise<Outcome<string>>>();
  /** The sessions whose stream the main process holds open: asked for once each. */
  private readonly followed = new Set<string>();

  constructor(private readonly api: DcodeApi) {}

  /** Starts listening to the main process; returns what stops it. */
  start(): () => void {
    const pending = new Set<ReturnType<typeof setTimeout>>();
    const offs = [
      this.api.onDaemon((daemon) => this.set({ daemon })),
      this.api.onConversations((raw) => this.onList(raw)),
      this.api.loops.on((push) => this.onLoops(push)),
      this.api.onSessionEvents((batch) => this.onEvents(batch.events)),
      this.api.onStreamEnd((end) => {
        // Ended for good: the next panel or open that wants it asks again.
        this.followed.delete(end.sessionId);
        // A conversation that ends closes its stream: that is the end the list
        // shows, not a failure. The list may say so a moment after the stream
        // does, over a connection of its own, so it is asked after that moment.
        // Gone from a list that has loaded is ended too.
        const timer = setTimeout(() => {
          pending.delete(timer);
          const c = this.snap.conversations.byId[end.sessionId];
          if (c ? !c.live : this.snap.conversations.loaded) return;
          const row = this.snap.rows.find((r) => r.id === end.sessionId);
          this.notice(`Os eventos de “${row?.title ?? end.sessionId}” pararam de chegar: ${end.reason}`);
        }, ENDED_GRACE_MS);
        pending.add(timer);
      }),
    ];
    return () => {
      offs.forEach((off) => off());
      pending.forEach((timer) => clearTimeout(timer));
    };
  }

  readonly subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  readonly getSnapshot = (): LiveSnapshot => this.snap;

  /** Registers what to call with each notice; returns what stops it. */
  readonly onNotice = (say: (text: string) => void): (() => void) => {
    this.noticed.add(say);
    return () => this.noticed.delete(say);
  };

  readonly actions: WindowActions = {
    open: (row) => (row.state === 'recorded' ? this.continueRow(row) : this.follow(row.id)),
    send: (id, text, steer, attachments) => this.send(id, text, steer, attachments ?? []),
    pickAttachments: async () => ({ ok: true, value: await this.api.pickAttachments() }),
    readAttachment: async (name, bytes) => outcome(await this.api.readAttachment(name, bytes), `${name} não foi anexado`),
    stop: async (id) => outcome(await this.api.interrupt(id), 'O turno não foi interrompido'),
    answer: async (id, approvalId, decision) =>
      outcome(await this.api.resolveApproval(id, approvalId, decision), 'A resposta não foi enviada'),
    newSession: async () => {
      const folder = await this.api.pickFolder();
      return folder ? this.newSessionIn(folder) : null;
    },
    newSessionIn: (workspace, options) => this.newSessionIn(workspace, options),
    pickFolder: () => this.api.pickFolder(),
    closeSession: async (id) => outcome(await this.api.closeSession(id), 'A sessão não fechou'),
    watch: (row) => {
      // An ended conversation has no stream: its panel reads the list.
      if (row.state === 'recorded' || this.followed.has(row.id)) return;
      void this.follow(row.id).then((out) => {
        // With the daemon down the bar already says so; one notice per panel would bury it.
        if (!out.ok && this.snap.daemon.state === 'connected') this.notice(out.why);
      });
    },
    listModels: async (workspace) => {
      const listed = await this.api.listModels(workspace);
      if (!listed.ok) return { ok: false, why: `O daemon não disse quais modelos existem: ${listed.refusal.message}` };
      const read = decodeModels(listed.value);
      return read.ok ? { ok: true, value: read.models } : { ok: false, why: `Resposta ilegível do daemon: ${read.reason}` };
    },
    switchModel: (id, workspace, model) => this.switchModel(id, workspace, model),
    setMode: async (id, mode) => outcome(await this.api.setMode(id, mode), 'O modo não mudou'),
    rename: async (id, name) => outcome(await this.api.renameConversation(id, name), 'A conversa não foi renomeada'),
    exec: async (id, command) => outcome(await this.api.exec(id, command), 'O comando não rodou'),
    undo: async (id) => decoded(outcome(await this.api.undo(id), 'O último turno não foi desfeito'), decodeUndo),
    compact: async (id) => decoded(outcome(await this.api.compact(id), 'O contexto não foi resumido'), decodeCompact),
    listSkills: async (workspace) =>
      read(await this.api.listSkills(workspace), decodeSkills, 'as skills de um projeto (GET /v1/skills)', 'O daemon não disse quais skills existem'),
    listCommands: async (workspace) =>
      read(await this.api.listCommands(workspace), decodeCommands, 'os comandos de um projeto (GET /v1/commands)', 'O daemon não disse quais comandos existem'),
    listMemory: async (workspace) =>
      read(await this.api.listMemory(workspace), decodeMemory, 'a memória de um projeto (GET /v1/memory)', 'O daemon não disse o que a memória tem'),
    readHistory: (id) =>
      readHistory(id, async (from) => {
        const page = await this.api.conversationEvents(id, from);
        if (page.ok) return page;
        const older = page.refusal.code === UNEXPECTED && / 404 /.test(page.refusal.message);
        return {
          ok: false,
          why: older
            ? notExposed('o histórico de uma conversa terminada (GET /v1/conversations/{id}/events)', page.refusal.message)
            : `O histórico desta conversa não foi lido: ${page.refusal.message}`,
        };
      }),
    readGit: async (id) => read(await this.api.readGit(id), decodeGit, 'o git da pasta (GET /v1/git)', 'O daemon não leu o git da pasta'),
    readGitDiff: async (id, path) =>
      read(await this.api.readGitDiff(id, path), decodeGitDiff, 'o diff de um arquivo (GET /v1/git/diff)', 'O daemon não deu o diff'),
    readGitPR: async (id) => prOutcome(await this.api.pr.read(id), 'o PR da branch (GET /v1/git/pr)', 'O PR da branch não foi lido'),
    openGitPR: async (id, draft) => prOutcome(await this.api.pr.open(id, draft), 'abrir um PR (POST /v1/git/pr)', 'O PR não foi aberto'),
    startLoop: async (workspace, args) => {
      const got = await this.api.loops.start(workspace, args);
      if (!got.ok) return { ok: false, why: startRefusal(got.refusal.code, got.refusal.message) };
      return read(got, decodeLoopRun, 'o /loop (POST /v1/loops)', 'O loop não começou');
    },
    stopLoop: async (id, when) => read(await this.api.loops.stop(id, when), decodeLoopRun, 'parar um loop (POST /v1/loops/{id}/stop)', 'O loop não parou'),
    continueLoop: async (id) => {
      const got = await this.api.loops.continue(id);
      if (!got.ok) return { ok: false, why: continueRefusal(got.refusal.code, got.refusal.message) };
      return read(got, decodeLoopRun, 'começar o trabalho de um loop (POST /v1/loops/{id}/continue)', 'O trabalho não começou');
    },
    openProposal: async (id, file) => outcome(await this.api.loops.openProposal(id, file), 'O arquivo da proposta não abriu'),
    listSpecs: async (workspace) =>
      read(await this.api.loops.specs(workspace), decodeSpecs, 'as pastas de spec de um projeto (GET /v1/specs)', 'O daemon não disse quais pastas de spec existem'),
  };

  private set(part: Partial<LiveSnapshot>): void {
    this.snap = { ...this.snap, ...part };
    for (const l of this.listeners) l();
  }

  private notice(text: string): void {
    for (const say of this.noticed) say(text);
  }

  private onList(raw: unknown): void {
    const decoded = decodeChange(raw);
    if (!decoded.ok) {
      this.notice(`Mudança ilegível na lista de conversas: ${decoded.reason}`);
      return;
    }
    const conversations = applyListChange(this.snap.conversations, decoded.change);
    this.set({ conversations, rows: Object.values(conversations.byId).map(rowOfConversation) });
  }

  /** A frame of the stream of loops, or why the daemon gives none — said, never taken for an empty list. */
  private onLoops(push: LoopsPush): void {
    if (push.kind === 'refused') {
      this.notice(push.reason);
      this.set({ loops: loopsUnavailable(this.snap.loops, push.reason) });
      return;
    }
    const decoded = decodeLoopChange(push.frame);
    if (!decoded.ok) {
      this.notice(`Mudança ilegível na lista de loops: ${decoded.reason}`);
      return;
    }
    this.set({ loops: applyLoopChange(this.snap.loops, decoded.change) });
  }

  private onEvents(raw: readonly unknown[]): void {
    let sessions = this.snap.sessions;
    const before = sessions.problems.length;
    for (const r of raw) sessions = applyDecoded(sessions, decodeEvent(r));
    for (const p of sessions.problems.slice(before)) this.notice(`Evento ilegível, sem sessão para mostrá-lo: ${p.reason}`);
    this.set({ sessions });
  }

  /**
   * A message is a turn for an idle session and a correction for a running one
   * (D23). Which one the window thinks it is comes from events that may still
   * be on their way; when the daemon answers that the session is the other
   * way, it is the daemon that knows, and the message goes the other way.
   *
   * Pictures (D46) and documents (D59) go with a turn only: a correction
   * carries text, so a message with attachments is never turned into one — it
   * is refused, said why, and kept for when the turn ends.
   */
  private async send(id: string, text: string, steer: boolean, attachments: Attachment[]): Promise<Outcome<null>> {
    if (steer && attachments.length > 0) return { ok: false, why: STEER_TAKES_TEXT };
    const first = steer ? await this.api.steer(id, text) : await this.api.submitTurn(id, text, attachments);
    const crossed = !first.ok && first.refusal.code === (steer ? CodeNoActiveTurn : CodeTurnAlreadyActive);
    if (!crossed) return outcome(first, steer ? 'A correção não foi enviada' : 'A mensagem não foi enviada');
    if (!steer && attachments.length > 0) return { ok: false, why: STEER_TAKES_TEXT };
    return steer
      ? outcome(await this.api.submitTurn(id, text), 'A mensagem não foi enviada')
      : outcome(await this.api.steer(id, text), 'A correção não foi enviada');
  }

  /**
   * Opens a session on what was chosen (D38). The mode goes as the sandbox it
   * maps to; the policy comes from the configuration, so when the session
   * opens in another mode it is switched at once. A refused switch leaves the
   * session open, as it opened, and says so.
   */
  private async newSessionIn(workspace: string, options: StartOptions = {}): Promise<Outcome<string>> {
    const asked: NewSessionOptions = {};
    if (options.model) asked.model = options.model;
    const sandbox = options.mode ? sandboxOf(options.mode) : undefined;
    if (sandbox) asked.sandboxMode = sandbox;
    const answer = Object.keys(asked).length > 0 ? await this.api.createSession(workspace, asked) : await this.api.createSession(workspace);
    const created = outcome(answer, `A sessão não abriu em ${workspace}`);
    if (!created.ok) return created;
    const id = created.value.id;
    const mode = modeToSwitch(created.value.mode ?? '', options.mode);
    if (mode) {
      const switched = await this.api.setMode(id, mode);
      if (!switched.ok) this.notice(`A sessão abriu, mas não no modo ${mode}: ${switched.refusal.message}`);
    }
    return this.follow(id);
  }

  private async follow(id: string): Promise<Outcome<string>> {
    if (this.followed.has(id)) return { ok: true, value: id };
    this.followed.add(id);
    const followed = outcome(await this.api.follow(id), 'A conversa não abriu');
    if (!followed.ok) this.followed.delete(id);
    return followed.ok ? { ok: true, value: id } : followed;
  }

  /**
   * A conversation changes model by continuing in a new session on it (D28).
   * The session it leaves is closed once the new one opens: left open, every
   * switch would keep one more, until the daemon refused to open any. Its
   * record stays, and the list shows it ended.
   */
  private async switchModel(id: string, workspace: string, model: string): Promise<Outcome<string>> {
    const made = outcome(await this.api.continueConversation({ id, workspace }, model), `A conversa não continuou em ${model}`);
    if (!made.ok) return made;
    // Live by the list; one the list does not know yet is live if its events are here.
    const live = this.snap.conversations.byId[id]?.live ?? id in this.snap.sessions.byId;
    if (live) {
      const closed = await this.api.closeSession(id);
      if (!closed.ok) this.notice(`A conversa continuou em ${model}, e a sessão anterior não fechou: ${closed.refusal.message}`);
    }
    return this.follow(made.value.id);
  }

  /**
   * An ended conversation is continued in a new session, which the window
   * opens (D21). One already continued opens its live continuation instead of
   * starting a second.
   */
  private continueRow(row: Row): Promise<Outcome<string>> {
    const live = liveContinuationOf(this.snap.conversations, row.id);
    if (live) return this.follow(live.id);
    const already = this.continuing.get(row.id);
    if (already) return already;
    const workspace = this.snap.conversations.byId[row.id]?.workspace ?? row.workspace;
    const started = (async (): Promise<Outcome<string>> => {
      const made = outcome(await this.api.continueConversation({ id: row.id, workspace }), 'A conversa não continuou');
      return made.ok ? this.follow(made.value.id) : made;
    })().finally(() => this.continuing.delete(row.id));
    this.continuing.set(row.id, started);
    return started;
  }
}
