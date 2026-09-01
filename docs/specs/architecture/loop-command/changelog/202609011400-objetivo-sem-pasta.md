# Objetivo sem pasta

**2026-09-01** — RN-8: `/loop <frase>` que não acha pasta de spec alguma abre a
sessão de qualificação em vez de recusar.

## O que acontecia

```
) /loop revise o projeto até entender
~ no specs/ folder here, or nothing in it. /loop <path> works on one folder.
```

O comando dizendo a alguém que o pedido dele era da forma errada — num produto
que tem uma família inteira para exatamente essa forma.

## A contradição

A `done-qualifier` foi escrita para o pedido em prosa. A `.r` dela nomeia o caso
que a motivou:

> O caso que motivou é o pedido em prosa. "Faça um cadastro de clientes" não traz
> `tasks.md` nenhum. (…) Falta a terceira resposta, que é a única construtiva:
> **se não há critério, levante um, meça-o, e me peça para assinar.**

E o `loopOne` — o caminho `/loop <caminho>` — já fazia isso:

```go
// loopOne works one spec folder, qualifying it first when it declares nothing.
if f.Criteria == 0 && f.Error == "" {
    q := spec; q.Qualify = true
    return p.loopSession(q)()
}
```

Só o caminho da frase morria antes de chegar lá. O comentário dele diz por quê:

> *"A sentence, not a path. **Find the spec folders**, keep the ones with work
> left, and work them one at a time."*

`/loop <frase>` foi construído para **escolher** entre pastas, não para aceitar
um briefing. Sem pastas, não havia o que escolher, e ele parou.

## A regra

Objetivo em prosa sem pasta alguma abre a sessão de qualificação sobre a própria
frase. Ela é o briefing inteiro, então é ela que o turno nomeia — e o turno não
manda ler especificação que não existe, porque procurar uma gasta as rodadas que
ele tem.

## Onde a proposta cai

`.dcode`, que é onde a definição de pronto de um workspace já mora e é uma das
três origens de `DoneSet`.

**Não** numa pasta derivada da frase. A RN-5 registra o defeito oposto — prosa
virando caminho, quando `/loop implemente todas as specs pendentes` foi procurar
`implemente/tasks.md` — e inventar `revise-o-projeto-ate-entender/` seria o
mesmo defeito de chapéu trocado.

`.dcode` é o **único** diretório que a escrita da proposta cria, e a restrição é
deliberada: caminho de spec que não existe continua sendo erro. Uma correção
mais larga quebrou o `TestACommitThatCannotWriteSaysWhere`, que guarda
exatamente isso desde antes de objetivo poder ser qualificado — pasta digitada
errada respondida com pasta criada é o silêncio que esta família recusa.

## Onde a constante mora

`protocol.WorkspaceDoneDir`. As duas metades precisam dela — o cliente decide
ancorar ali, o daemon escreve ali — e antes disso era literal em dois lugares,
dos quais só um estava certo sobre para que servia.

## Um teste que era teatro, evitado

O primeiro teste desta mudança exercitava `GoalToQualify` direto. Ele passava
com a ligação no handler **removida**: helper testado e ignorado pelo chamador é
a forma que passa enquanto o produto continua recusando. O teste que vale passa
pelo `Update`, que é onde a recusa era desenhada, e foi visto vermelho sem a
ligação.

## Invariantes

- `TestAGoalWithNoSpecFolderIsQualified`
- `TestTheHandlerDivertsAGoalWithNoFoldersInsteadOfRefusing`
- `TestTheQualifyingTurnForAGoalNamesTheSentence`
- `TestTheHandlerStillDrawsThePlanWhenFoldersExist`
- `TestAnEmptyGoalIsNotQualified`
