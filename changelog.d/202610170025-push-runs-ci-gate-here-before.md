## en

- **A push runs the CI gate here before it leaves.** `.githooks/pre-push`,
  enabled with `make hooks`, runs on the pushed commit, in a throwaway
  worktree, what CI runs: the Go gate when anything outside `desktop/` changed,
  the desktop gate when `desktop/` or `internal/protocol/` did. Seven pull
  request runs failed in one night, one of them on gofmt, which a local gate
  catches in a second; the gate was a rule nothing enforced.

## pt-BR

- **Um push roda aqui o gate da CI antes de sair.** `.githooks/pre-push`,
  ligado com `make hooks`, roda no commit enviado, num worktree descartável, o
  que a CI roda: o gate de Go quando algo fora de `desktop/` mudou, o do
  desktop quando mudou `desktop/` ou `internal/protocol/`. Sete execuções de
  PR falharam numa noite, uma delas por gofmt, que um gate local pega em um
  segundo; o gate era uma regra que nada fazia cumprir.
