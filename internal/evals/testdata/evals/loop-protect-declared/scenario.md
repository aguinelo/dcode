# loop-protect-declared

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **100%**

`protected` declarado no frontmatter e `--protect` no argumento são união.

## Por que 100% é legítimo

Porque não depende do modelo. `TestLoadSpecWithProtectLayersBoth` afirma que os
dois entram e `TestLoadSpecProtectIsNotDuplicated` que a mesma glob vinda das
duas fontes aparece uma vez.

União, não precedência: `Protected` destaca no relatório em vez de proibir, e
"o arquivo vence o argumento" abriria a única direção que não pode existir por
acidente — um argumento removendo uma proteção que o arquivo pediu.
