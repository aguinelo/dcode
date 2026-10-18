## en

- **A merge waits for the person.** `internal/orchestrate` gains the merge component. The
  integration place is a detached worktree of the daemon's own for each project, never the
  person's tree. In it, a child's branch is brought up to date by merging main into it, never by
  rewriting it. A conflict goes back to the child with its paths and the markers left in the
  child's worktree; the orchestrator resolves none of it. On GitHub each child becomes a PR with a
  fixed block of evidence, and never the gate's output, which may hold a secret. The merge is a
  squash with `--match-head-commit` on the measured commit. With no forge, it is a squash and a
  conditional update of main, refused while main is checked out anywhere, and `forge.local` says
  what that loses. Step by step, nothing merges before the person answers `merge`. `return` and
  `close` are `merge.refused` with the person's note. The queue is serial: a main that moved is
  `gate_stale`, and a red, pending or unknown main is `main_red` for every merge but a fix.
  `main.health` is read after each merge until CI ends or `orchestrator.ci_timeout` passes.
  `orchestrator.forge` (`DCODE_ORCHESTRATOR_FORGE`), `orchestrator.ci_timeout`,
  `orchestrator.sensitive` and `orchestrator.sensitive_diff_lines` are now read. This is the core
  of step 10 of the orchestrator's first slice. Calling it from each child's pipeline comes with
  the wiring of step 7.

## pt-BR

- **Um merge espera a pessoa.** `internal/orchestrate` ganha o componente de merge. O lugar de
  integração é uma worktree destacada do próprio daemon para cada projeto, nunca a árvore da
  pessoa. Nele a branch do filho se atualiza com o merge da main, sem reescrever nada. Conflito
  volta ao filho com os caminhos e os marcadores na worktree dele, e a orquestradora não resolve
  nada. No GitHub cada filho vira um PR com um bloco fixo de evidência, e nunca a saída do gate,
  que pode ter um segredo. O merge é um squash com `--match-head-commit` no commit medido. Sem
  forge, é um squash e uma troca condicional da main, recusada com a main em checkout em qualquer
  lugar, e `forge.local` diz o que se perde. Em passo a passo, nada faz merge antes da resposta
  `merge` da pessoa. `return` e `close` são `merge.refused` com a nota dela. A fila é serial: main
  que andou é `gate_stale`, e main vermelha, pendente ou desconhecida é `main_red` para todo merge
  que não seja um conserto. `main.health` é lida depois de cada merge até o CI terminar ou
  `orchestrator.ci_timeout` passar. `orchestrator.forge` (`DCODE_ORCHESTRATOR_FORGE`),
  `orchestrator.ci_timeout`, `orchestrator.sensitive` e `orchestrator.sensitive_diff_lines` passam a
  ser lidas. É o núcleo do passo 10 da primeira fatia do orquestrador. Chamá-lo na esteira de cada
  filho vem com a fiação do passo 7.
