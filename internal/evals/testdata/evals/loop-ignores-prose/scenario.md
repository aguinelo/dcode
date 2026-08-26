# loop-ignores-prose

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **≥ 99%** (aspiracional)

`/loop` parser **não pode** interpretar prosa como critério. Texto narrativo
("smoke manual", "validar com o usuário") vira `CriterionUnavailable` no
relatório final, **não** `Criterion{Name: "smoke"}`.

## Status: aspiracional

Mesma situação de `loop-parses-spec`: a medição real exige a tool
`loop_load`, que ainda não existe. Cobertura determinística em
`internal/loop/loopcommand/loopspec_test.go::TestLoadSpecIgnoresProse`.
