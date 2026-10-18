# Passo tardio não desfaz fechamento

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §7, §8, invariantes; `.i` passo 6).

## O que aconteceu

`TestCreatingChildrenLeavesTheOrchestratorIdleWhileTheyRun` falhou sob carga — 3 vezes em 800
execuções com a suíte inteira rodando ao lado — com o filho fechado pelo `DELETE` da pessoa ainda
listado em `/children`, como `failed`/`incomplete`. O teste esperava o fim do turno do filho
(`Reason` preenchido pelo `child.completed`) e fechava. Mas o `child.completed` é só o primeiro
commit do fim de turno: logo depois, `afterDone` vê que nada foi commitado e move o filho para
`failed`. `move` lia o estado, conferia que não era terminal e só então fazia o commit, que
escrevia o estado novo sem olhar de novo. Quando o `Close` comitava `closed` nesse intervalo, o
`failed` tardio passava por cima, e um filho fechado voltava à lista como falho — com a worktree
já removida.

O mesmo vão existia no próprio `Close` (lê, fecha a sessão, comita `closed` por cima de um
`merged` que tenha chegado enquanto isso), em `commitWaiting`, no `child.completed`, na devolução
(`child.returned`) e na passagem para `pr`.

## O que mudou

- `commitLive` (`children.go`): a conferência de que o filho não está fechado nem mergeado é feita
  sob o mesmo lock do commit, não quando quem chama leu o filho. Passo que perde a corrida não
  escreve linha no log nem estado.
- `move`, `Close`, `commitWaiting`, o `child.completed` de `turnEnded`, `giveBack` e a passagem para
  `pr` passam por ele. `Close` que perde para um merge responde `invalid_input` dizendo o estado em
  que o filho ficou; passagem para `pr` que perde para um fechamento diz à pessoa que a pergunta
  de merge ficou sobre um filho que já não existe.

## Por que o teste novo não depende de tempo

`TestALateStepDoesNotUndoAMergeOrACloseThatLandedFirst` põe o merge dentro do `Close` do
spawner falso — depois de o `Close` ler o filho e antes do commit dele —, então a ordem é a do
código, não a do agendador. O teste de integração que falhava continua igual: o que ele prova
(fechado e sem retenção, some da lista) era certo; errada era a produção.

## O deadlock que apareceu no gate

No gate da primeira versão desta mudança, `TestScenarioARedMainStopsTheQueue` ficou 9 minutos
parado. Duas ordens de lock opostas: `Merges.Answer` segurava `m.mu` enquanto emitia
`question.answered` (e, num `return`, `merge.refused`), e o emit chega a `o.mu` pela
`Project()`; `saveMerges`, no `drain` da fila, segurava `o.mu` dentro do `apply` e lia
`mg.Queued()`/`mg.Main()`, que pedem `m.mu`. A inversão já existia; a fila e a resposta só
precisavam se cruzar.

- `Merges.Answer` toma a resposta sob o lock (`take`) e emite com ele solto, como `ask` e a saúde
  da main já faziam. "A primeira resposta vence" continua decidido sob o lock.
- `saveMerges` lê perguntas, fila e saúde da main antes de tomar `o.mu`.
- `TestAnAnswerIsSaidWithTheMergesLockReleased`: quem ouve o evento pergunta de volta aos merges;
  antes, esperava a própria resposta até o teto do teste, sempre — sem depender de agendador.
