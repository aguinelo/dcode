# loop-parses-spec

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **100%**

`tasks.md` bem-formado vira a `DoneSet` do golden, na ordem do arquivo.

## Por que 100% é legítimo

Porque não depende do modelo. É o parser, e o parser é determinístico. A
asserção está em três lugares: `TestLoadSpecHappyPath` para a forma,
`TestLoadSpecPreservesOrder` para a ordem e `TestLoadSpecSeparatorIsNotSyntax`
para o que **não** é contrato — a pontuação entre o número da tarefa e o
`verify:`, que numa primeira versão exigia travessão literal e fazia um arquivo
inteiro escrito com hífen voltar como zero critérios e nenhum erro.

O material existe para o ID não sumir da tabela — um contrato sem fixture é um
contrato que a guarda não consegue casar.
