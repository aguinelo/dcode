// What the person does with an orchestrator from the window (D66): answer a
// question waiting in the queue, and open a project's orchestrator. Pure: the
// words the window says when the daemon did not take it, from its code and
// its own message — never a reason guessed from the message.

import type * as P from '../protocol/generated';
import { CodeOrchestratorUnavailable, CodeQuestionAlreadyAnswered, CodeQuestionNotFound, RoleChild, RoleOrchestrator, RoleReviewer } from '../protocol/generated';

/** How long a note on an answer may be, in characters: a sentence or a paragraph, not a document. */
export const NOTE_LIMIT = 2000;

/**
 * Whether the answer takes a note: a merge or a design question, where the
 * note says why — returned with what missing, closed why. The ceiling's two
 * answers say all there is (continue or stop).
 */
export function takesNote(q: P.Question): boolean {
  return q.kind !== 'ceiling';
}

/** An answer the daemon did not take, said as the window says it. */
export function answerRefusal(code: string, message: string): string {
  switch (code) {
    case CodeQuestionAlreadyAnswered:
      // The first answer wins (.p §5.1): another window, the TUI or another person got there first.
      return `Esta pergunta já foi respondida — por outra janela, pela TUI ou por outra pessoa —, e a primeira resposta é a que vale: ${message}`;
    case CodeQuestionNotFound:
      return `A orquestradora não tem mais esta pergunta: ${message}`;
    default:
      return `A resposta não foi enviada: ${message}`;
  }
}

/**
 * A project's orchestrator that did not open. The orchestrator is opt-in in
 * this slice, and it needs the sessions recorded to come back after a
 * restart: unavailable is said with the keys that turn it on, beside the
 * daemon's own reason, which names the one that is off.
 */
export function openRefusal(code: string, message: string, workspace: string): string {
  const base = `A orquestradora de ${workspace} não abriu: ${message}`;
  if (code !== CodeOrchestratorUnavailable) return base;
  return `${base}. Para ligar: orchestrator.enabled = true e record.enabled = true na configuração do dcode (ou DCODE_ORCHESTRATOR_ENABLED=true), e reinicie o daemon.`;
}

/**
 * How a conversation of the list opens. An orchestrator is never continued:
 * it is the live session of its project, opened by the project's route (RN-2),
 * live or recorded — a new ordinary session would have neither its role nor
 * its tools. A child or a reviewer that ended is read, never continued, for
 * the same reason. Any other conversation that ended is continued (D21, D33).
 */
export type OpenPath = 'follow' | 'orchestrator' | 'ended-agent' | 'continue';

export function openPathOf(row: { state: string; role?: string }): OpenPath {
  if (row.role === RoleOrchestrator) return 'orchestrator';
  if (row.state !== 'recorded') return 'follow';
  return row.role === RoleChild || row.role === RoleReviewer ? 'ended-agent' : 'continue';
}

/** Whether a conversation is one an orchestrator conducts — itself, a child or a reviewer. */
export function isAgentRole(role: string | undefined): boolean {
  return role === RoleOrchestrator || role === RoleChild || role === RoleReviewer;
}

/** A child or a reviewer that ended, written to or opened to continue. */
export const ENDED_AGENT =
  'Este agente terminou. Um agente da orquestradora não continua numa sessão nova, que não teria o papel nem as ferramentas dele: o histórico fica aqui para ler, e o trabalho segue pela orquestradora do projeto.';

/** An orchestrator the list does not place in a project. */
export const ORCHESTRATOR_WITHOUT_PROJECT =
  'Esta orquestradora não diz de que projeto é, e uma orquestradora não continua numa sessão nova: abra-a por "Nova" › "◆ orquestradora", na pasta do projeto.';

/** A model switch is a continuation (D28), which an orchestrator's sessions never have. */
export const NO_MODEL_SWITCH =
  'Uma sessão da orquestradora não troca de modelo: trocar é continuar a conversa numa sessão nova, que não teria o papel nem as ferramentas dela.';

/** session_not_found, said in words: the daemon's own is "no session <id>". */
export function sessionGone(what: string): string {
  return `${what}: esta sessão não está mais no daemon — ela terminou, ou o daemon reiniciou sem ela. Abra a conversa de novo pela lista.`;
}
