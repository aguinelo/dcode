## en

- **A session can have a parent.** `Session` and `Conversation` gain `parent_id`, `project_id`,
  `role` (`orchestrator`, `child` or `reviewer`; `ValidRole` refuses anything else) and `owns`,
  all omitted when empty, so a session nobody orchestrates is the same bytes as before. The list
  of conversations reads the four from `session.created`, so a recorded child still says whose it
  was after the daemon that ran it is gone. `internal/protocol` gains the orchestrator's types
  (`Project`, `Child`, `Question`, the gate's and the reviewer's results), the ten `child.*`,
  `review.verdict`, `gate.measured`, `question.*`, `merge.*` and `main.health` events, and the
  thirteen error codes, each with its HTTP status; `pkg/client` gains the seven calls of the
  orchestrator's routes. No route answers yet: this is step 2 of the first slice, putting the
  vocabulary on the wire so the steps after it change behaviour only. `TurnStarted.Origin`,
  `OriginChildren` and `StopDaemonRestart` come with step 3, which reopens a recorded session
  under its own id and needs them to compile.

## pt-BR

- **Uma sessão pode ter mãe.** `Session` e `Conversation` ganham `parent_id`, `project_id`,
  `role` (`orchestrator`, `child` ou `reviewer`; `ValidRole` recusa o resto) e `owns`, todos
  omitidos quando vazios, então a sessão que ninguém orquestra tem os bytes de antes. A lista de
  conversas lê os quatro do `session.created`, e um filho gravado continua dizendo de quem é
  depois que o daemon que o rodou se foi. `internal/protocol` ganha os tipos da orquestradora
  (`Project`, `Child`, `Question`, os resultados do gate e do revisor), os dez eventos `child.*`,
  `review.verdict`, `gate.measured`, `question.*`, `merge.*` e `main.health`, e os treze códigos
  de erro, cada um com o status HTTP; `pkg/client` ganha as sete chamadas das rotas da
  orquestradora. Nenhuma rota responde ainda: é o passo 2 da primeira fatia, que põe o
  vocabulário no fio para os passos seguintes mudarem só comportamento. `TurnStarted.Origin`,
  `OriginChildren` e `StopDaemonRestart` entram com o passo 3, que reabre uma sessão gravada sob o
  próprio id e não compila sem eles.
