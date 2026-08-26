# loop-parses-spec

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **≥ 99%** (aspiracional)

`/loop` lê `tasks.md` e produz a `DoneSet` que o engine consome.

## Status: aspiracional

Esta fixture existe para satisfazer o guard `TestEveryDeclaredContractHasItsFixture`
em `internal/evals/contracts_test.go`: o `.p` declara um caminho e o caminho
precisa existir com `task.md` carregável. O conteúdo do `task.md` é
placeholder porque a medição real exige a tool `loop_load`, que **não
existe ainda no produto** — esta spec está em `experimental`.

A medição correta virá quando a tool existir. Até lá:

- O **parser** é coberto deterministicamente por
  `internal/loop/loopcommand/loopspec_test.go` (TDD, golden file).
- O **contrato comportamental** declarado no `.p §7` permanece, com o
  threshold intacto.
- A **promoção para `stable`** (definida em `loop-command.p §1`) exige que
  esta fixture vire material real antes de `make eval` rodar o threshold
  contra o modelo.
