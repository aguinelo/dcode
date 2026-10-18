## en

- **A child the person closed stays closed.** A step of the child's pipeline
  read the child, checked it was neither closed nor merged, and only then
  committed — writing its state over whatever had landed in between. A close
  that landed in that window was undone: the child came back to `/children`
  as failed, its worktree already gone (3 failures in 800 runs of
  `TestCreatingChildrenLeavesTheOrchestratorIdleWhileTheyRun` with the whole
  suite as load). A close likewise wrote over a merge that landed while it
  closed the session. The check is now made under the lock of the commit
  itself, for every transition, the end of a turn, a hand-back and the move to
  `pr`; a step that loses writes nothing, and a close that loses to a merge
  says so.

## pt-BR

- **Um filho que a pessoa fechou continua fechado.** Um passo do pipeline do
  filho lia o filho, conferia que não estava fechado nem mergeado e só então
  comitava — escrevendo seu estado por cima do que tivesse chegado no meio. Um
  fechamento que caísse nesse intervalo era desfeito: o filho voltava a
  `/children` como falho, com a worktree já removida (3 falhas em 800
  execuções de `TestCreatingChildrenLeavesTheOrchestratorIdleWhileTheyRun` com
  a suíte inteira como carga). Do mesmo jeito, um fechamento escrevia por cima
  de um merge que chegasse enquanto fechava a sessão. A conferência agora é
  feita sob o lock do próprio commit, em toda transição, no fim de turno, na
  devolução e na passagem para `pr`; o passo que perde não escreve nada, e o
  fechamento que perde para um merge diz isso.
