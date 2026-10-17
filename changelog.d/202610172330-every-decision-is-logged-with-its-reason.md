## en

- **Every decision is logged with its reason.** The new `internal/orchestrate` package starts with
  the orchestrator's decision log: `OpenDecisionLog` opens
  `<orchestrator.dir>/<project-id>/decisions.jsonl`, owner-only, and `Record` writes one JSON line,
  syncs it, and only then returns its `seq`. A decision without a `why`, of a kind outside the
  spec's list, or for another project is refused and nothing is written; a write that fails is an
  error, is taken back and spends no `seq`, so the transition it decided does not happen instead of
  happening unlogged. Reopening continues the `seq`; a last line a crash tore is cut and handed back
  by `Torn` to be said, and a complete line that does not read or a `seq` out of order keeps the log
  closed, naming the line. A line over 64 KiB cuts its largest free-text field and says how many
  bytes it lost. `orchestrator.dir` (`DCODE_ORCHESTRATOR_DIR`, default `<state>/orchestrate`,
  absolute) is now read. Nothing opens the log yet: this is step 4 of the orchestrator's first
  slice, and the log comes before the orchestrator because every step after it writes there first.

## pt-BR

- **Toda decisão vai ao log com o porquê.** O pacote novo `internal/orchestrate` começa pelo log de
  decisões da orquestradora: `OpenDecisionLog` abre `<orchestrator.dir>/<project-id>/decisions.jsonl`,
  só do dono, e `Record` escreve uma linha JSON, faz `fsync` e só então devolve o `seq`. Decisão sem
  `why`, de `kind` fora da lista da spec, ou de outro projeto é recusada e nada é escrito; escrita
  que falha é erro, é desfeita e não gasta `seq`, então a transição que ela decidia não acontece em
  vez de acontecer sem registro. Reabrir continua o `seq`; a última linha que uma queda rasgou é
  cortada e devolvida por `Torn` para ser dita, e linha completa ilegível ou `seq` fora de ordem
  mantém o log fechado, nomeando a linha. Linha acima de 64 KiB corta o maior campo de texto e diz
  quantos bytes perdeu. `orchestrator.dir` (`DCODE_ORCHESTRATOR_DIR`, padrão
  `<state>/orchestrate`, absoluto) passa a ser lida. Ninguém abre o log ainda: é o passo 4 da
  primeira fatia do orquestrador, e o log vem antes da orquestradora porque todo passo depois dele
  escreve ali primeiro.
