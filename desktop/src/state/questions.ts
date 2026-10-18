// What the person does with an orchestrator from the window (D66): answer a
// question waiting in the queue, and open a project's orchestrator. Pure: the
// words the window says when the daemon did not take it, from its code and
// its own message — never a reason guessed from the message.

import type * as P from '../protocol/generated';
import { CodeOrchestratorUnavailable, CodeQuestionAlreadyAnswered, CodeQuestionNotFound } from '../protocol/generated';

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
