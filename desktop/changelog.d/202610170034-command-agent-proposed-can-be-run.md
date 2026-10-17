## en

- **A command the agent proposed can be run by the person, in a terminal below
  the conversation, and its result handed back.** Asked for after a denied
  approval the person wanted to run themselves, possibly with sudo. Every
  proposed command — the approval card's and each `bash` line in the flow —
  gets Copiar and Rodar no terminal, shown on hover or focus and out of the
  layout, so both visual references stay at 0.00%. Rodar no terminal docks a
  real shell (a PTY in the main process, `node-pty`, drawn by xterm.js) below
  the flow — in the grid window, the maximized conversation and Crew's thread —
  in the session's workspace as the daemon lists it, with the command typed and
  not run: the person edits it and presses Enter, and sudo asks for its
  password inside the terminal. zsh and bash mark their prompts (OSC 133), so
  when the command ends the window offers the result to the agent, showing the
  message first — command, exit code, the last 200 lines without ANSI, cut and
  said so, password-prompt lines dropped — and sends it only on a click, as a
  turn or a correction. Named channels checked in the main process; each
  terminal answers only the window that opened it; nothing of its stream is
  logged. Dependencies, exact: `node-pty` 1.2.0-beta.15 (1.1.0 ships macOS's
  spawn-helper without its exec bit), `@xterm/xterm` 6.0.0, `@xterm/addon-fit`
  0.11.0, loaded only when a terminal opens (D37).

## pt-BR

- **Um comando que o agente propôs pode ser rodado pela pessoa, num terminal
  abaixo da conversa, e o resultado volta ao agente.** Pedido depois de uma
  aprovação negada que a pessoa queria rodar ela mesma, talvez com sudo. Todo
  comando proposto — o do cartão de aprovação e cada linha `bash` do fluxo —
  ganha Copiar e Rodar no terminal, que aparecem ao passar o ponteiro ou com o
  foco e ficam fora do layout, e as duas referências visuais seguem a 0,00%.
  Rodar no terminal abre um shell de verdade (um PTY no processo principal,
  `node-pty`, desenhado pelo xterm.js) abaixo do fluxo — na janela da grade, na
  conversa maximizada e na conversa da Crew —, na pasta da sessão como o daemon
  a lista, com o comando digitado e não rodado: a pessoa edita e aperta Enter,
  e o sudo pede a senha dentro do terminal. zsh e bash marcam o prompt (OSC
  133), e quando o comando termina a janela oferece o resultado ao agente,
  mostrando antes a mensagem — comando, código de saída, as últimas 200 linhas
  sem ANSI, cortadas e dito que foram, sem as linhas de prompt de senha — e só
  manda com um clique, como turno ou correção. Canais nomeados conferidos no
  processo principal; cada terminal responde só à janela que o abriu; nada do
  que passa por ele vai a log. Dependências, em versões exatas: `node-pty`
  1.2.0-beta.15 (a 1.1.0 publica o spawn-helper do macOS sem permissão de
  execução), `@xterm/xterm` 6.0.0, `@xterm/addon-fit` 0.11.0, carregados só
  quando um terminal abre (D37).
