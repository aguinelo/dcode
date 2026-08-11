# no-budget-noise-when-low

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **100%**

Sessão curta, bem abaixo da primeira faixa; **nenhum** lembrete de orçamento.

## Por que 100% é legítimo

Porque não depende do modelo. Nada abaixo do primeiro limiar emite, e isso é
asserção em dois lugares: `TestAShortSessionCrossesNothing` na camada pura e
`TestNoBudgetReminderOnAShortSession` no laço.

O material existe para o ID não sumir da tabela — um contrato sem fixture é um
contrato que a guarda não consegue casar, e foi assim que trinta deles ficaram
inertes.
