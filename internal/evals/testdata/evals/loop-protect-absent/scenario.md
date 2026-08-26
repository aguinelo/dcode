# loop-protect-absent

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **100%**

Sem declaração, nenhum caminho é protegido.

## Por que 100% é legítimo

Porque não depende do modelo. `TestLoadSpecWithoutProtectDeclaresNothing`
afirma a ausência, e `TestLoadSpecFrontmatterEdges` afirma que uma lista vazia,
uma lista de vazios e um frontmatter com outras chaves também produzem nada.

É a RN-4 da família: o harness não decide o que é medição, e a posição da spec
não é declaração. Inferir `protected` de `specs/**` seria o harness escolhendo,
por conta própria, o que o agente não pode tocar.
