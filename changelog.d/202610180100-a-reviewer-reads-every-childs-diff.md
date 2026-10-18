## en

- **A reviewer reads every child's diff.** `internal/orchestrate` gains the review: `AssembleReview`
  builds what the reviewer reads from the child's worktree — the task, what it owns, the specs the
  task names and the diff `Base..Head` — and a diff that touches a path outside what the child owns
  is `outside_owns`, handed back without asking any model; `Uncommitted` and `OutsideOwns` make the
  same check on `git status` before a commit. A diff past `orchestrator.review_max_diff_bytes` is
  left out whole and said, never truncated into something that reads as the whole of it. The
  reviewer's registry is `read`, `glob`, `grep`, `symbol` and the new `verdict` tool — nothing that
  writes. `verdict` ends the review with `approve` or `changes_requested` and its findings, and
  `review.verdict` goes to the decision log with the reviewer's why; a review that ends without a
  verdict records `missing` and is an error, never an approval. `orchestrator.reviewer_profiles`
  (`DCODE_ORCHESTRATOR_REVIEWER_PROFILES`, an allow-list: a profile off it is `model_not_allowed`)
  and `orchestrator.review_max_diff_bytes` (`DCODE_ORCHESTRATOR_REVIEW_MAX_DIFF_BYTES`, `131072`,
  from `4096` to `1048576`) are now read. It is the core of step 8 of the orchestrator's first
  slice; spawning the reviewer and calling it from each child's pipeline come with the wiring.

## pt-BR

- **Um revisor lê o diff de cada filho.** `internal/orchestrate` ganha a revisão: `AssembleReview`
  monta o que o revisor lê a partir da worktree do filho — a tarefa, o `owns`, as specs que a
  tarefa nomeia e o diff `Base..Head` —, e diff que toca caminho fora do `owns` é `outside_owns`,
  devolvido sem consultar modelo nenhum; `Uncommitted` e `OutsideOwns` fazem a mesma conferência
  sobre o `git status` antes de um commit. Diff acima de `orchestrator.review_max_diff_bytes` sai
  inteiro e é dito, nunca truncado em algo que se leia como o todo. O registro do revisor é `read`,
  `glob`, `grep`, `symbol` e a ferramenta nova `verdict` — nada que escreva. `verdict` termina a
  revisão com `approve` ou `changes_requested` e os achados, e `review.verdict` vai ao log de
  decisões com o porquê do revisor; revisão que termina sem veredito grava `missing` e é erro,
  nunca aprovação. `orchestrator.reviewer_profiles` (`DCODE_ORCHESTRATOR_REVIEWER_PROFILES`, lista
  permitida: perfil fora dela é `model_not_allowed`) e `orchestrator.review_max_diff_bytes`
  (`DCODE_ORCHESTRATOR_REVIEW_MAX_DIFF_BYTES`, `131072`, de `4096` a `1048576`) passam a ser lidas.
  É o núcleo do passo 8 da primeira fatia do orquestrador; criar o revisor e chamá-lo na esteira de
  cada filho vêm com a fiação.
