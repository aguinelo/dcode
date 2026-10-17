## en

- **A pure reducer from protocol events to what the window shows**, tested with
  vitest. It reads only fields the protocol carries, validated at the boundary: a
  payload that does not match is said in the session's flow, a repeated event
  changes nothing, a missing one is named. Where the design asks for what the
  wire does not carry — a child's progress, live tokens, the approval's requested
  paths, the loop — it draws what exists and `docs/DECISIONS.md` lists the gap,
  because an invented field is a promise the daemon never made.

## pt-BR

- **Um redutor puro de eventos do protocolo para o que a janela mostra**, testado
  com vitest. Lê só campos que o protocolo traz, validados na fronteira: payload
  que não bate é dito no fluxo da sessão, evento repetido não muda nada, evento
  que falta é nomeado. Onde o design pede o que o fio não traz — o progresso de
  um filho, tokens ao vivo, os caminhos pedidos na aprovação, o loop —, desenha o
  que existe e o `docs/DECISIONS.md` lista a lacuna, porque campo inventado é
  promessa que o daemon nunca fez.
