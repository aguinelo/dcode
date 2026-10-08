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

// What the Crew sections say where the daemon does not expose the data yet
// (D33): said, never an empty list that reads as "none".
/** How every "this daemon does not expose it" starts, so the section can tell it from a failure. */
export const NOT_EXPOSED = 'Este daemon ainda não expõe';

/** A daemon older than the route (N6): said with the daemon's own answer. */
export function notExposed(what: string, said: string): string {
  return `${NOT_EXPOSED} ${what} — atualize o dcode. O daemon disse: ${said}`;
}
export const ROUTINES_NOT_YET = 'O daemon ainda não lista as rotinas — a agenda vem com as rotinas no núcleo.';
export const APPS_NOT_YET = 'O dcode ainda não conecta apps externos — esta seção fica vazia até o núcleo ter conectores.';
export const KNOWLEDGE_NOT_YET = 'O dcode ainda não tem base de conhecimento além da memória do projeto — esta seção fica vazia até o núcleo ter uma.';
