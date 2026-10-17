## en

- **The orchestrator lists its agents in the TUI.** In a session with `role: orchestrator` the
  side column's top half is the live list of its children instead of the diff: each with its
  state among the eight, tokens in, out and cache, its PR and CI, and what it waits on, under a
  count of the questions waiting for the person and main's health. The list is a fold over the
  orchestrator's own log (`child.*`, `question.*`, `review.verdict`, `gate.measured`, `merge.*`,
  `main.health`), never the text of a turn; the daemon's `/children` answer replaces it once the
  client reads it on opening, with the rest of the step. A closed or merged
  child leaves when the daemon stops listing it — the retention is the daemon's — and a failed one
  stays. The orchestrator never writes, so its diff pane would have been empty for good. An
  unreadable event is said in the stream, never folded as nothing. `dcode orchestrate`, `^Q` and
  `^R` over the children come with the rest of the step.

## pt-BR

- **A orquestradora lista os agentes na TUI.** Numa sessão com `role: orchestrator`, a metade de
  cima da coluna lateral é a lista viva dos filhos em vez do diff: cada um com o estado entre os
  oito, tokens de entrada, saída e cache, PR e CI, e o que espera, sob a contagem das perguntas
  que esperam a pessoa e a saúde da main. A lista é uma dobra do log da própria orquestradora
  (`child.*`, `question.*`, `review.verdict`, `gate.measured`, `merge.*`, `main.health`), nunca
  do texto de um turno; a resposta de `/children` do daemon a substitui quando o cliente a ler ao
  abrir, com o resto do passo. Fechado ou merged sai quando o
  daemon para de listá-lo — a retenção é do daemon —, e `failed` fica. A orquestradora nunca
  escreve, então o painel de diff dela ficaria vazio para sempre. Evento ilegível é dito no fluxo,
  nunca dobrado como nada. `dcode orchestrate`, `^Q` e `^R` sobre os filhos vêm com o resto do
  passo.
