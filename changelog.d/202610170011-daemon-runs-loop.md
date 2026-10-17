## en

- **The daemon runs `/loop`.** `POST /v1/loops` takes the argument as typed
  and the daemon conducts the sequence the TUI ran in its own process: survey
  first, qualify a folder that declares nothing, stop at a written proposal
  (`awaiting_review`) until `POST /v1/loops/{id}/continue`, work a goal's
  pending folders one session each, and end with the verdict of the whole run.
  `GET /v1/loops` and the `GET /v1/loops/events` stream show every run, and
  `POST /v1/loops/{id}/stop` stops now or after the current folder. One live
  loop per workspace (`loop_already_running`); sessions and conversations a
  loop opened carry `loop_id`; the Go client gains `StartLoop`, `ListLoops`,
  `WatchLoops`, `ContinueLoop` and `StopLoop`. The sequence and the queue lived
  in the TUI's memory, so a second client could neither start a loop nor see
  one, and reimplementing it would be a second copy of the rule. The TUI is
  unchanged here and still runs its own loop until it moves to these routes.
  A turn that ends in error or at a ceiling fails the run with the reason
  rather than counting its folder done, a stop leaves no session working, and
  a run that only qualified waits for continue where the TUI waits to be typed
  again.

## pt-BR

- **O daemon roda o `/loop`.** `POST /v1/loops` recebe o argumento como foi
  digitado e o daemon conduz a sequência que a TUI rodava no próprio processo:
  pergunta primeiro, qualifica a pasta que não declara nada, para na proposta
  escrita (`awaiting_review`) até `POST /v1/loops/{id}/continue`, trabalha as
  pendentes de um objetivo uma sessão por vez, e termina com o veredito da
  corrida inteira. `GET /v1/loops` e o fluxo `GET /v1/loops/events` mostram
  todo loop, e `POST /v1/loops/{id}/stop` para agora ou depois da pasta
  corrente. Um loop vivo por workspace (`loop_already_running`); sessões e
  conversas que um loop abriu carregam `loop_id`; o cliente Go ganha
  `StartLoop`, `ListLoops`, `WatchLoops`, `ContinueLoop` e `StopLoop`. A
  sequência e a fila viviam na memória da TUI, então um segundo cliente não
  conseguia começar nem ver um loop, e reimplementá-la seria uma segunda cópia
  da regra. A TUI não muda aqui e segue rodando o próprio loop até passar a
  usar estas rotas. Turno que termina em erro ou num teto faz o loop falhar
  com o motivo em vez de contar a pasta como pronta, parar não deixa sessão
  trabalhando, e um loop que só qualificou espera o continue onde a TUI espera
  o /loop ser digitado de novo.
