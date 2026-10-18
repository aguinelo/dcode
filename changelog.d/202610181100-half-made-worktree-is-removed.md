## en

- **A half-made worktree is removed, and a stopped gate says it was stopped.**
  A caller that cancelled while the gate's throwaway worktree was being made
  killed `git worktree add` half-way, which leaves the worktree locked
  ("initializing"); the cleanup's single `--force` refuses a locked tree, so
  the worktree stayed behind and the error was git's kill, not the
  cancellation. This turned main's CI red on macOS. Worktrees are now removed
  with `remove -f -f`, the directory and `prune`, on a context of their own,
  for the gate, a child's worktree, a failed add and the integration place,
  and what still stays is said. A stopped gate returns the cancellation, with
  git's message and any cleanup failure beside it. The pre-push hook removes
  its tree the same way. Decision `orchestration/changelog/202610181100-worktree-meio-feita-sai.md`.

## pt-BR

- **Worktree meio feita sai, e gate parado diz que foi parado.** Quem chamava
  e cancelava enquanto a worktree descartável do gate era feita matava o
  `git worktree add` no meio, que deixa a worktree trancada ("initializing");
  o `--force` único da limpeza recusa worktree trancada, então ela ficava, e o
  erro era a morte do git, não o cancelamento. Isso deixou o CI da main
  vermelho no macOS. Worktrees agora saem com `remove -f -f`, o diretório e
  `prune`, num contexto próprio, no gate, na worktree do filho, no `add` que
  falha e no lugar de integração, e o que ainda fica é dito. Gate parado
  devolve o cancelamento, com a mensagem do git e a falha da limpeza ao lado.
  O hook de pre-push remove a sua árvore do mesmo jeito. Decisão `orchestration/changelog/202610181100-worktree-meio-feita-sai.md`.
