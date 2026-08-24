# A cauda tem dois pisos

**Data:** 2026-08-25
**Specs afetadas:** `context-engine` (`.p`, seção 7)
**Fonte:** pedido de quem usa — "dar uma boa reduzida mantendo 30% final intacto"

## O que mudou

`ce.Config` ganhou `KeepFraction`, padrão `0.30`. A cauda preservada passou a
ter **dois pisos**, e vence o que proteger mais.

## Contagem é a unidade errada

`KeepTurns: 4` conta turnos. Turnos variam em uma ordem de grandeza: uma
pergunta de uma linha e uma investigação de quarenta ferramentas são ambas um
turno.

Quatro turnos curtos protegem quase nada — o resumo come a investigação inteira
e mantém quatro "ok". Quatro turnos longos não deixam o que compactar.

A fração diz **quanto** da conversa sobrevive; a contagem diz **quantas trocas**
sobrevivem. As duas respondem perguntas diferentes, e por isso as duas ficam.

## Medido com a mesma estimativa

`protectedByTokens` usa o mesmo `Estimate` que o gatilho usa, sobre as mesmas
mensagens. Dois estimadores para uma pergunta é como uma regra dispara num
tamanho que a outra nunca viu — que é exatamente o defeito que o medidor de
contexto acabou de ter.

## A RN-6 continua acima dos dois

A `RoleUser` mais recente e tudo depois dela nunca entram no trecho compactado,
digam o que disserem os dois pisos. A tarefa corrente sobrevive por construção.

## Sem janela, sem fração

Se ninguém reportou a janela, não há denominador, e a contagem decide sozinha.
Proteger o histórico inteiro nesse caso pararia a compactação de acontecer —
que é pior do que compactar por uma regra só.

## Um caso que eu escrevi e não existia

A primeira versão do teste exercitava "muitos turnos pequenos numa janela de um
milhão". Ali o histórico inteiro cabe em 30% da janela, então a fração protege
tudo — e o teste falhou.

Só que esse estado é **inalcançável**: `Plan` só faz essa pergunta depois de o
contexto já estar em 80% da janela. Se o histórico todo cabe em 30%, ninguém
está compactando. O teste foi refeito com uma janela que o histórico de fato
enche.
