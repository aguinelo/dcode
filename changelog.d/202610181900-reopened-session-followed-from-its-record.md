## en

- **A reopened or retention-trimmed session is followed from any `from` its record holds.**
  `GET /v1/sessions/{id}/events?from=N` below what the session keeps in memory now reads that part
  from the session's record and goes on live from there, each event once — what replaying already
  did. The stream refused it with `events_expired`, and a session reopened under its own id (#519)
  keeps nothing in memory, so after a restart the desktop said the events of the orchestrator and of
  every child had stopped arriving. With recording off, the answer is still `410 events_expired`.

## pt-BR

- **Uma sessão reaberta ou aparada pela retenção é acompanhada de qualquer `from` que o registro
  guarde.** `GET /v1/sessions/{id}/events?from=N` abaixo do que a sessão guarda em memória passa a
  ler essa parte do registro da sessão e seguir ao vivo dali, cada evento uma vez — o que a
  reposição já fazia. O fluxo recusava com `events_expired`, e uma sessão reaberta sob o próprio id
  (#519) não guarda nada em memória; depois de um reinício, o desktop dizia que os eventos da
  orquestradora e de cada filho tinham parado de chegar. Com a gravação desligada, a resposta
  continua `410 events_expired`.
