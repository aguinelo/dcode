// What the person is asked before the app stops a daemon it started while
// sessions are at work or loops are alive (D19, D55): how many stop, in the
// window's language. A loop lives in the daemon, not in a client, so it ends
// with the daemon and does not come back. The question defaults to Cancelar —
// a ↵ must not cut a turn off.

export interface QuitQuestion {
  message: string;
  detail: string;
  /** Fechar first, Cancelar second: the index the dialog answers with. */
  buttons: [string, string];
}

const ENDS = 'O daemon foi aberto pelo DCode Box e fecha com ele';

export function quitQuestion(working: number, loops = 0): QuitQuestion {
  const sessions = working === 1 ? '1 sessão' : `${working} sessões`;
  const are = working === 1 ? 'Uma sessão está' : `${working} sessões estão`;
  const runs = loops === 1 ? '1 loop' : `${loops} loops`;
  const alive = loops === 1 ? 'um loop está vivo' : `${loops} loops estão vivos`;
  const lasts = loops === 1 ? 'um loop não sobrevive ao daemon: ele termina junto' : 'um loop não sobrevive ao daemon: eles terminam junto';
  const buttons: [string, string] = ['Fechar', 'Cancelar'];
  if (loops === 0) {
    return {
      message: `Fechar o DCode Box e parar ${sessions}?`,
      detail: `${are} rodando ou esperando aprovação. ${ENDS}, e as sessões dele param junto.`,
      buttons,
    };
  }
  if (working === 0) {
    return {
      message: `Fechar o DCode Box e encerrar ${runs}?`,
      detail: `${alive.charAt(0).toUpperCase()}${alive.slice(1)}. ${ENDS}, e ${lasts}, e não volta quando o app abrir de novo.`,
      buttons,
    };
  }
  return {
    message: `Fechar o DCode Box, parar ${sessions} e encerrar ${runs}?`,
    detail: `${are} rodando ou esperando aprovação, e ${alive}. ${ENDS}: as sessões param, e ${lasts}.`,
    buttons,
  };
}
