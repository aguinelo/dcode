// The window connected: the daemon's status, its list of conversations and the
// events of the sessions this window opened, kept as one snapshot the window
// draws. Everything arrives from the main process through the preload's API;
// nothing here touches a socket (D2).

import { decodeChange } from '../protocol/conversations';
import { decodeEvent } from '../protocol/events';
import { decodeModels } from '../protocol/models';
import { CodeNoActiveTurn, CodeTurnAlreadyActive } from '../protocol/generated';
import type { Answer, DaemonStatus, DcodeApi } from '../shared/api';
import { applyListChange, emptyConversations, liveContinuationOf, type ConversationsState } from '../state/conversations';
import { applyDecoded, emptySessions, type SessionsState } from '../state/sessions';
import { rowOfConversation, type Row } from '../state/sidebar';
import type { Outcome, WindowActions } from './window';

export interface LiveSnapshot {
  daemon: DaemonStatus;
  conversations: ConversationsState;
  /** The sidebar's rows, derived from the list. */
  rows: Row[];
  /** The sessions this window follows, folded from their events. */
  sessions: SessionsState;
}

/** How long the list has to show a conversation ended before its stream's end is called a failure. */
const ENDED_GRACE_MS = 1500;

/** A refusal said as the window says it: what did not happen, then the daemon's reason. */
function outcome<T>(answer: Answer<T>, what: string): Outcome<T> {
  return answer.ok ? { ok: true, value: answer.value } : { ok: false, why: `${what}: ${answer.refusal.message}` };
}

export class LiveStore {
  private snap: LiveSnapshot = {
    daemon: { state: 'connecting' },
    conversations: emptyConversations,
    rows: [],
    sessions: emptySessions,
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
    send: (id, text, steer) => this.send(id, text, steer),
    stop: async (id) => outcome(await this.api.interrupt(id), 'O turno não foi interrompido'),
    answer: async (id, approvalId, decision) =>
      outcome(await this.api.resolveApproval(id, approvalId, decision), 'A resposta não foi enviada'),
    newSession: async () => {
      const folder = await this.api.pickFolder();
      return folder ? this.newSessionIn(folder) : null;
    },
    newSessionIn: (workspace) => this.newSessionIn(workspace),
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
   */
  private async send(id: string, text: string, steer: boolean): Promise<Outcome<null>> {
    const first = steer ? await this.api.steer(id, text) : await this.api.submitTurn(id, text);
    const crossed = !first.ok && first.refusal.code === (steer ? CodeNoActiveTurn : CodeTurnAlreadyActive);
    if (!crossed) return outcome(first, steer ? 'A correção não foi enviada' : 'A mensagem não foi enviada');
    return steer
      ? outcome(await this.api.submitTurn(id, text), 'A mensagem não foi enviada')
      : outcome(await this.api.steer(id, text), 'A correção não foi enviada');
  }

  private async newSessionIn(workspace: string): Promise<Outcome<string>> {
    const created = outcome(await this.api.createSession(workspace), `A sessão não abriu em ${workspace}`);
    return created.ok ? this.follow(created.value.id) : created;
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
