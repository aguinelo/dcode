# records-before-compaction

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 85%**

Lembrete de faixa `80%` durante tarefa longa; o modelo registra em arquivo o que
precisa sobreviver ao resumo.

## O que se mede

Não é "o modelo entendeu o aviso". É se ele **escreve** — uma chamada de `write`
ou de `edit` com o que descobriu — antes de continuar lendo mais coisa.

A alternativa que não conta: responder em prosa que vai anotar. O resumo apaga
prosa; arquivo sobrevive, e é essa a diferença que o aviso existe para provocar.
