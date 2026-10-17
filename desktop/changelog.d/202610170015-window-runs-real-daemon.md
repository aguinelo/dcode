## en

- **The window runs on a real daemon.** Inside Electron, the main process
  attaches to the `dcode serve` answering on the socket — `DCODE_SOCKET`, else
  what `dcode socket` prints — or starts one there as its child, from
  `DCODE_BIN`, `~/.local/bin/dcode` or the `PATH`; it stops only the one it
  started, and asks first when quitting would stop a session that is running or
  waiting for you (D19). The bottom bar says connecting, connected with the
  daemon's version, lost or failed, with the reason. The sidebar is the daemon's
  list of conversations (N3), live and ended, by project: a live one opens its
  stream from the first event and resumes from the last when it drops; an ended
  one is continued in a new session the window opens (D21). The field sends a
  turn, corrects a running one (D23) and stops it; an approval is answered with
  `1`, `2`, `3`, `esc` and `↵`, which denies (D22); Nova sessão and ⌘N ask for a
  folder and open a session there. Every refusal shows the daemon's message, and
  nothing sent to a daemon that died looks accepted. The renderer still sees
  neither Node nor the socket: the preload hands it one named channel per request
  and per notice (`src/shared/api.ts`). A browser, or `?fixture=`, still shows
  the recording, and the visual check reads 1.20% and 1.21% — the numbers D22 and
  D23 were measured at.

## pt-BR

- **A janela roda sobre um daemon de verdade.** Dentro do Electron, o processo
  principal anexa ao `dcode serve` que responde no socket — `DCODE_SOCKET`, senão
  o que o `dcode socket` imprimir — ou sobe um ali como filho, de `DCODE_BIN`,
  `~/.local/bin/dcode` ou do `PATH`; encerra só o que subiu, e pergunta antes
  quando fechar pararia uma sessão rodando ou esperando você (D19). A barra
  inferior diz conectando, conectado com a versão do daemon, caiu ou não subiu,
  com o porquê. A lateral é a lista de conversas do daemon (N3), vivas e
  terminadas, por projeto: uma viva abre o fluxo dela desde o primeiro evento e o
  retoma do último quando ele cai; uma terminada é continuada numa sessão nova,
  que a janela abre (D21). O campo envia um turno, corrige um que roda (D23) e o
  para; uma aprovação se responde com `1`, `2`, `3`, `esc` e `↵`, que nega (D22);
  Nova sessão e ⌘N pedem uma pasta e abrem uma sessão nela. Toda recusa mostra a
  mensagem do daemon, e nada enviado a um daemon que morreu parece aceito. O
  renderer continua sem ver Node nem o socket: o preload lhe entrega um canal com
  nome por pedido e por notícia (`src/shared/api.ts`). Um navegador, ou o
  `?fixture=`, continua mostrando a gravação, e a régua visual lê 1,20% e 1,21% —
  os números em que D22 e D23 foram medidas.
