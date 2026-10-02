// What the person is asked before the app stops a daemon it started while
// sessions are at work (D19): how many stop, in the window's language. The
// question defaults to Cancelar — a ↵ must not cut a turn off.

export interface QuitQuestion {
  message: string;
  detail: string;
  /** Fechar first, Cancelar second: the index the dialog answers with. */
  buttons: [string, string];
}

export function quitQuestion(working: number): QuitQuestion {
  const sessions = working === 1 ? '1 sessão' : `${working} sessões`;
  const are = working === 1 ? 'Uma sessão está' : `${working} sessões estão`;
  return {
    message: `Fechar o DCode e parar ${sessions}?`,
    detail: `${are} rodando ou esperando aprovação. O daemon foi aberto pelo DCode e fecha com ele, e as sessões dele param junto.`,
    buttons: ['Fechar', 'Cancelar'],
  };
}
