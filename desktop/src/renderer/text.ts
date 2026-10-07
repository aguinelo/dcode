// The words the window says about the stream. Portuguese, as the design's
// texts are; one place, so a second language is one file and not a hunt.

import { sealOf, type Tone } from '../state/flow';
import { plural } from '../state/format';
import type { Note } from '../state/session';

const STOPPED: Record<string, string> = {
  max_iterations: 'Parou no teto de rodadas',
  repeat_loop: 'Parou: a mesma chamada se repetiu',
  max_tokens: 'Parou no teto de tokens',
  interrupted: 'Interrompido',
};

export function noteText(n: Note): { text: string; tone: Tone } {
  switch (n.kind) {
    case 'resumed':
      return { text: `Continua a sessão ${n.sourceId} · ${plural(n.turns, 'turno', 'turnos')}`, tone: 'dim' };
    case 'notice':
      // The daemon's sentence, in English, as session.error's is: it carries
      // counts and names this side cannot rebuild from the code.
      return { text: n.message, tone: 'warn' };
    case 'mode':
      return { text: `Modo ${n.previous} → ${n.mode} · ${n.sandbox}`, tone: 'dim' };
    case 'compacted':
      return {
        text: n.messages > 0 ? `Contexto resumido: ${n.messages} mensagens viraram resumo, ${n.kept} ficaram` : 'Contexto resumido',
        tone: 'dim',
      };
    case 'skill':
      return { text: n.whenToUse ? `Skill ${n.name} entrou no turno — ${n.whenToUse}` : `Skill ${n.name} entrou no turno`, tone: 'dim' };
    case 'context':
      return { text: `Contexto em ${Math.round(n.fraction * 100)}% do caminho até o resumo`, tone: 'dim' };
    case 'completion':
      return sealOf(n.completion) ?? { text: '', tone: 'dim' };
    case 'stopped': {
      const base = STOPPED[n.reason] ?? `Parou: ${n.reason}`;
      const rounds = n.reason === 'max_iterations' && n.maxRounds ? ` (${n.rounds} de ${n.maxRounds})` : '';
      return { text: `${base}${rounds}`, tone: n.reason === 'interrupted' ? 'dim' : 'warn' };
    }
    case 'done-proposed':
      return {
        text: `Definição de pronto proposta · ${plural(n.criteria, 'critério', 'critérios')}${n.noAcceptance ? ' · nenhum vermelho' : ''}`,
        tone: 'dim',
      };
    case 'done-signed':
      return { text: 'Definição de pronto assinada', tone: 'dim' };
    case 'gap':
      return { text: `Faltam eventos ${n.from}–${n.to} desta sessão: o que aconteceu ali não está na tela`, tone: 'err' };
    case 'orphan-result':
      return { text: `Chegou o resultado de uma chamada que não foi anunciada (${n.callId})`, tone: 'err' };
    case 'protocol':
      return { text: `Evento ilegível: ${n.reason}`, tone: 'err' };
  }
}

/** What the window says when an action needs the daemon this version does not reach. */
export const NOT_CONNECTED = 'Sem daemon: esta janela mostra eventos gravados, e nada foi enviado.';

export function notYet(what: string): string {
  return `${what} ainda não existe nesta versão.`;
}
