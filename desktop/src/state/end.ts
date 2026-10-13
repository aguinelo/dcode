// Ending a session from the ⋯ menu (D51). The daemon closes it
// (`DELETE /v1/sessions/{id}`) and keeps its record, so the conversation stays
// in the list, ended, and writing in it continues it in a new session (D21).

/** What ending now would cut short, said before it is confirmed; null when nothing would. */
export function endWarning(state: string | undefined): string | null {
  if (state === 'running') return 'Um turno está rodando: encerrar o interrompe no meio e fecha a sessão.';
  if (state === 'blocked') return 'Uma aprovação espera resposta: encerrar interrompe o turno e fecha a sessão.';
  return null;
}

/** Said once the daemon closed it: where the conversation went. */
export const ENDED = 'Sessão encerrada. A conversa fica na lista, terminada; escrever nela continua numa sessão nova.';
