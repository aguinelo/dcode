## en

- **The orchestrator runs the gate itself.** `internal/orchestrate` gains the gate: `Gate.Measure`
  runs the project's declared command on one exact commit, in a throwaway worktree made for that run
  alone — never the person's tree, never a child's — with every `GIT_*` variable out of the
  environment, a timeout that kills the whole process group, the last 4 KiB of output in the result
  and the whole of it in `gates/<commit>.log`, and the worktree removed on return, on a timeout and
  on a panic. `RecordGate` writes `gate.measured` with the measurement and, beside it as a `claim`,
  what the child said of its own gate, which never changes the result. `CheckFresh` is the merge's
  lock: a measurement of another commit, or over another main, is `gate_stale`. No command declared
  is `gate_undeclared`, and nothing runs. `orchestrator.gate_command`
  (`DCODE_ORCHESTRATOR_GATE_COMMAND`, falling back to `verify.command`) and
  `orchestrator.gate_timeout` (`DCODE_ORCHESTRATOR_GATE_TIMEOUT`, `30m`, from `1m` to `6h`) are now
  read. It is the core of step 9 of the orchestrator's first slice, the pre-push gate brought inside;
  calling it from each child's pipeline, after review and before the merge queue, comes with steps 6
  to 8.

## pt-BR

- **A orquestradora roda o gate ela mesma.** `internal/orchestrate` ganha o gate: `Gate.Measure`
  roda o comando que o projeto declara num commit exato, numa worktree descartável feita só para
  aquela medição — nunca a árvore da pessoa, nunca a de um filho —, com todo `GIT_*` fora do
  ambiente, um tempo-limite que mata o grupo de processos inteiro, os últimos 4 KiB da saída no
  resultado e a saída inteira em `gates/<commit>.log`, e a worktree removida ao voltar, no
  `timeout` e num pânico. `RecordGate` escreve `gate.measured` com a medição e, ao lado dela como
  `claim`, o que o filho disse do próprio gate, que nunca muda o resultado. `CheckFresh` é a trava
  do merge: medição de outro commit, ou sobre outra main, é `gate_stale`. Sem comando declarado é
  `gate_undeclared`, e nada roda. `orchestrator.gate_command` (`DCODE_ORCHESTRATOR_GATE_COMMAND`,
  com `verify.command` como queda) e `orchestrator.gate_timeout` (`DCODE_ORCHESTRATOR_GATE_TIMEOUT`,
  `30m`, de `1m` a `6h`) passam a ser lidas. É o núcleo do passo 9 da primeira fatia do
  orquestrador, o gate do pre-push trazido para dentro; chamá-lo na esteira de cada filho, depois da
  revisão e antes da fila de merge, vem com os passos 6 a 8.
