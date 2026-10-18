## en

- **A child works in its own worktree.** The orchestrator gains `delegate`, `children`,
  `steer_child` and `close_child`. `delegate` is all or nothing: owns that are empty, absolute, not
  clean or outside the tree, owns that cross each other or a live child's (`owns_overlap`, both
  paths named), a batch past `orchestrator.max_children` (`max_children_reached`), a model outside
  `orchestrator.child_profiles` (`model_not_allowed`) and a project past
  `orchestrator.token_ceiling` (`ceiling_reached`) refuse the whole batch before any worktree is
  made, said in the decision log and to the person. Each child is a recorded session with the child
  role, its parent and its owns, in a git worktree under `orchestrator.dir` — never the person's
  tree — on a branch `<orchestrator.branch_prefix><slug>-<id6>` from main's tip; `delegate` comes
  back as soon as the first turns start, and the orchestrator stays idle while they run. A child's
  writes outside its owns are refused (`outside_owns`), git that writes is refused in its shell
  without asking, with the reason, its approvals wait for the person without a deadline, and at the
  end of each of its turns the harness commits what it changed, adding the paths by name.
  `child.started`, `child.state` and `child.completed` reach the orchestrator's wire,
  `GET /v1/sessions/{id}/children` answers (a closed child leaves it after
  `orchestrator.closed_retention`), and `DELETE` of a child is the person's close: worktree removed,
  owns freed. Every session of a project counts its tokens against the ceiling; reaching it stops
  dispatching, says why, and asks a `ceiling` question whose `continue` opens a new window. Children
  come back under their own ids when the daemon starts again; one whose turn was open fails with
  `daemon_restart`. Step 6 of the orchestrator's first slice.

## pt-BR

- **Um filho trabalha na própria worktree.** A orquestradora ganha `delegate`, `children`,
  `steer_child` e `close_child`. O `delegate` é tudo ou nada: `owns` vazio, absoluto, sujo ou fora
  da árvore, `owns` que se cruzam entre si ou com filho vivo (`owns_overlap`, com os dois caminhos),
  lote acima de `orchestrator.max_children` (`max_children_reached`), modelo fora de
  `orchestrator.child_profiles` (`model_not_allowed`) e projeto acima de
  `orchestrator.token_ceiling` (`ceiling_reached`) recusam o lote inteiro antes de qualquer
  worktree, ditos no log de decisões e à pessoa. Cada filho é uma sessão gravada com o papel de
  filho, a mãe e o `owns`, numa worktree git sob `orchestrator.dir` — nunca a árvore da pessoa —, na
  branch `<orchestrator.branch_prefix><slug>-<id6>` a partir da ponta da main; o `delegate` volta
  assim que os primeiros turnos começam, e a orquestradora fica `idle` com eles rodando. Escrita do
  filho fora do `owns` é recusada (`outside_owns`), git que grava é negado no `bash` dele sem
  perguntar e com o porquê, as aprovações dele esperam a pessoa sem prazo, e ao fim de cada turno o
  harness commita o que mudou, adicionando os caminhos pelo nome. `child.started`, `child.state` e
  `child.completed` chegam ao fio da orquestradora, `GET /v1/sessions/{id}/children` responde
  (fechado sai depois de `orchestrator.closed_retention`), e o `DELETE` de um filho é o fechar da
  pessoa: worktree removida, `owns` liberado. Toda sessão do projeto conta tokens contra o teto;
  chegar nele para o despacho, diz por quê e faz uma pergunta `ceiling` cujo `continue` abre outra
  janela. Os filhos voltam sob o próprio id quando o daemon reinicia; o que tinha turno aberto vai a
  `failed` com `daemon_restart`. Passo 6 da primeira fatia do orquestrador.
