## en

- **The stop-reason guard checks itself against the protocol, and three
  working-defaults invariants are claimed.** `TestEveryStopReasonIsReachableFromTheLoop`
  kept its list by hand and had left out `StopRefused`, which the loop returns;
  it now reads every `Stop*` the protocol declares and fails on one it does not
  classify, and `StopDaemonRestart` is classified as what session reopen
  returns, never the loop. Behaviour-definition §8 gains five lines for what
  the working-defaults spec listed as unclaimed — the practices line of the
  doctrine audit, an empty probe, and an unreadable manifest named and warned
  about — each claimed by the test that already asserted it. The audit test
  now checks the `Practices` line it never looked for.

## pt-BR

- **A guarda dos motivos de parada se confere contra o protocolo, e três
  invariantes da working-defaults ganham dono.** `TestEveryStopReasonIsReachableFromTheLoop`
  mantinha a lista à mão e tinha deixado de fora `StopRefused`, que o laço
  devolve; agora lê todo `Stop*` que o protocolo declara e reprova o que não
  classifica, e `StopDaemonRestart` fica classificado como devolvido pela
  reabertura da sessão, nunca pelo laço. A §8 da behavior-definition ganha
  cinco linhas para o que a spec working-defaults listava sem reivindicação — a
  linha de práticas da auditoria da doutrina, a sonda vazia e o manifesto
  ilegível nomeado e avisado —, cada uma reivindicada pelo teste que já a
  afirmava. O teste da auditoria passa a conferir a linha `Practices`, que
  nunca procurou.
