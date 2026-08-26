# loop-protect-absent

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **≥ 99%** (aspiracional)

`/loop` **não** infere `Protected` da posição da spec. Quando `tasks.md` não
declara `protected` no frontmatter e nenhuma flag `--protect` é passada,
`Protected` da `DoneSet` é vazio. **Nenhum path** é protegido por default.

## Status: aspiracional

Mesma situação dos outros contracts desta spec: a tool `loop_load` ainda
não existe. Cobertura determinística em
`internal/loop/loopcommand/loopspec_test.go::TestLoadSpecHappyPath` (caso
sem `protected` no frontmatter) e `TestLoadSpecZeroCriteriaIsNotAnError`.
