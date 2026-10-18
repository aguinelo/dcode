# A spec segue o código

**Data:** 2026-10-18
**Specs afetadas:** `202608080016-behavior-definition` (`.p`, `.config`)

> **Regra:** correção de documentação. Nenhum comportamento muda; a spec passa
> a dizer o que `internal/behavior` já faz.

## O que estava errado

- **Lembretes que não existem, e os que existem sem estar escritos.** O bloco
  `ReminderKind` da seção 5 declarava `verification_stale` e
  `verification_failed`, que o código nunca teve, e um tipo `Verification` que
  não mora neste pacote. Faltavam seis que o código emite:
  `instruction_out_of_chain`, `unmet_criteria`, `worth_remembering`,
  `protected_touched`, `interrupted` e `cycle_undone`. O bloco foi reescrito a
  partir de `internal/behavior/reminders.go`. Uma nota diz onde o estado de
  verificação vive de fato (`loop.Verification`, em `internal/loop/done.go`) e
  por que ao modelo ele chega como `unmet_criteria` ou
  `verification_unavailable`: a verificação é a definição de pronto de um item
  só (`agent-loop/202608110930`). `interrupted` é dito como emitido pelo loop,
  fora de `Emit`.
- **A assinatura de `Build`.** Era `Build(p Prompt, family provider.Family)`;
  é `Build(p Prompt, f Formulation)`. A interface `Formulation` e
  `FormulationFor` entram no `.p`, e com elas o motivo do desenho: a interface
  só sabe delimitar seção, então não tem como mudar uma regra. `Prompt` ganhou
  os campos que tem — `Tools` como nomes, `Repo`, `Workspace`, `Boundary`.
- **Os índices de changelog.** O `.config` listava duas entradas de vinte, cada
  uma duas vezes; o `.p` repetia `202608262330` e não listava onze. Os dois
  índices agora listam todas, em ordem.

## O que não mudou

Nenhum texto de lembrete, nenhuma chave, nenhuma invariante. As linhas da
seção 8 do `.p` ficam como estavam, e com elas as reivindicações em
`internal/behavior/invariants_test.go`.
