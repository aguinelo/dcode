## en

- **A recorded session reopens under its own id.** `session.Manager.Reopen` brings a recorded
  session back with the same id, the history rebuilt from its record, the sequence continuing from
  the last recorded event without a gap, and the file opened for appending; a turn the record shows
  open is ended with `turn.completed{reason: daemon_restart}` before anything new. Continuing a
  conversation opens a new session that replays it, which is right for `dcode -c` and wrong for the
  orchestrator: its project and every child name it by id, and a daemon restart must not orphan
  them. A live session, a missing or pruned record and a damaged one are refused and said, never
  answered with a fresh session under the old id. With it, `SubmitOrigin` stamps
  `TurnStarted.Origin`, and `session.Approver` asks without a deadline when its timeout is zero.
  Nothing calls the reopen yet; the daemon does, at startup, in a later step.

## pt-BR

- **A sessão gravada reabre sob o próprio id.** `session.Manager.Reopen` traz de volta uma sessão
  gravada com o mesmo id, o histórico reconstruído do registro, a sequência continuando do último
  evento gravado sem buraco, e o arquivo aberto em acréscimo; turno que o registro mostra aberto
  termina com `turn.completed{reason: daemon_restart}` antes de qualquer coisa nova. Continuar uma
  conversa abre uma sessão nova que a reproduz, certo para `dcode -c` e errado para a
  orquestradora: o projeto e cada filho a nomeiam pelo id, e um reinício do daemon não pode
  deixá-los órfãos. Sessão viva, registro ausente ou podado e registro danificado são recusados e
  ditos, nunca respondidos com uma sessão nova sob o id antigo. Junto, `SubmitOrigin` marca
  `TurnStarted.Origin`, e `session.Approver` pergunta sem prazo quando o tempo dele é zero. Ninguém
  chama a reabertura ainda; o daemon chama, no boot, num passo seguinte.
