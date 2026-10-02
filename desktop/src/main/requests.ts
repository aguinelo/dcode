import { Version, type CreateSessionRequest, type Session } from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import { UNREACHABLE, type Answer, type DaemonStatus, type Refusal } from '../shared/api';
import { UNEXPECTED, request, shown } from './wire';

// The window's requests to the daemon, one route each. Every one ends in an
// Answer, and none is sent while the daemon is not connected: saying so, with
// the reason the status bar shows, is the answer — a message typed to a dead
// daemon must never look accepted.

export function sessionPath(id: string): string {
  return `/${Version}/sessions/${encodeURIComponent(id)}`;
}

/** Why nothing was sent: where the window stands with the daemon. */
export function notConnected(status: DaemonStatus): Refusal {
  switch (status.state) {
    case 'connecting':
      return { code: UNREACHABLE, message: 'O app ainda está procurando o daemon, e nada foi enviado.' };
    case 'connected':
      return { code: UNREACHABLE, message: 'O daemon está conectado.' };
    case 'lost':
      return { code: UNREACHABLE, message: `O daemon caiu, e nada foi enviado. ${status.reason}` };
    case 'failed':
      return { code: UNREACHABLE, message: `Não há daemon, e nada foi enviado. ${status.reason}` };
  }
}

export class Requests {
  constructor(private readonly status: () => DaemonStatus) {}

  /** Opens a session in a folder, with the daemon's defaults. */
  createSession(workspace: string): Promise<Answer<Session>> {
    return this.open({ workspace });
  }

  /** Continues a recorded conversation in a new session (D21). */
  continueConversation(id: string, workspace: string): Promise<Answer<Session>> {
    return this.open({ workspace, resume: id });
  }

  submitTurn(sessionId: string, text: string): Promise<Answer<null>> {
    return this.command(`${sessionPath(sessionId)}/turns`, { text });
  }

  /** A correction for the turn running (D23). */
  steer(sessionId: string, text: string): Promise<Answer<null>> {
    return this.command(`${sessionPath(sessionId)}/steer`, { text });
  }

  interrupt(sessionId: string): Promise<Answer<null>> {
    return this.command(`${sessionPath(sessionId)}/interrupt`);
  }

  resolveApproval(sessionId: string, approvalId: string, decision: string): Promise<Answer<null>> {
    return this.command(`${sessionPath(sessionId)}/approvals/${encodeURIComponent(approvalId)}`, { decision });
  }

  private async call(method: string, path: string, body?: unknown): Promise<Answer<unknown>> {
    const now = this.status();
    if (now.state !== 'connected') return { ok: false, refusal: notConnected(now) };
    return request(now.socket, method, path, body);
  }

  private async open(body: CreateSessionRequest): Promise<Answer<Session>> {
    const a = await this.call('POST', `/${Version}/sessions`, body);
    if (!a.ok) return a;
    // The window opens what it was answered: a session it cannot name is one it cannot open.
    if (!isRecord(a.value) || typeof a.value.id !== 'string' || a.value.id === '') {
      return { ok: false, refusal: { code: UNEXPECTED, message: `O daemon respondeu à abertura da sessão sem o id dela: ${shown(a.value)}` } };
    }
    return { ok: true, value: a.value as unknown as Session };
  }

  private async command(path: string, body?: unknown): Promise<Answer<null>> {
    const a = await this.call('POST', path, body);
    return a.ok ? { ok: true, value: null } : a;
  }
}
