# loop-protect-declared

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **≥ 95%** (aspiracional)

`/loop` carrega `Protected` do frontmatter de `tasks.md` (ou da flag
`--protect`). Quando o agente toca caminho protegido durante o turno, a
mudança é **destacada** no relatório (`TouchedProtected`), nunca contada
como progresso em silêncio.

## Status: aspiracional

Mesma situação dos outros contracts desta spec: a tool `loop_load` ainda
não existe. Cobertura determinística em
`internal/loop/loopcommand/loopspec_test.go::TestLoadSpecWithProtectLayersBoth`.
