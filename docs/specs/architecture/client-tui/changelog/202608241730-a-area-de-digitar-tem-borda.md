# A área de digitar tem borda

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** pedido do usuário — "coloque delimitador claro na área de digitação"

## O que mudou

A linha de digitar virou uma área com moldura nos quatro lados, com uma coluna
de calha dentro.

```
┌──────────────────────────────────────────────────────────────────────────┐
│ > primeira linha                                                         │
│   segunda linha                                                 12 acima │
└──────────────────────────────────────────────────────────────────────────┘
```

## Moldura aqui e não em volta de uma chamada

Hoje mesmo a borda completa em volta de cada chamada de ferramenta foi
**rejeitada com medida**, e esta não contradiz aquilo — é o que distingue as
duas.

A área de digitar é um **campo**: região fixa, que não rola, à qual se volta, e
que precisa ser encontrada sem ler. Uma chamada de ferramenta é **conteúdo**, e
moldura em volta de conteúdo é moldura em volta do que já se estava lendo.

O custo aqui é limitado e conhecido: duas linhas da tela e duas colunas de uma
região que tem uma a três linhas. O custo lá era duas linhas e duas colunas
**por chamada**, mais a borda entrando na seleção do modo de cópia.

## A moldura não carrega estado

A primeira versão a deixava âmbar enquanto a linha tinha o teclado e apagada
enquanto o fluxo tinha, para tornar visível o estado que estava por trás do bug
do `v`.

O teste que eu escrevi para ela fez a pergunta óbvia seguinte: **essa distinção
sobrevive sem cor?** Não sobrevivia. Era cor e nada mais, que é a única coisa que
um indicador de estado aqui não pode ser.

Então a moldura parou de carregar estado, e o estado ficou onde já é desenhado:
na entrada sob o cursor do fluxo. O que a moldura responde é a pergunta que não
tem outra resposta na tela — **onde vão as letras que eu digito** — e essa
resposta não muda.

## Duas alturas eram uma

`BodyHeight` subtraía `InputRows`, que conta só as linhas de texto. Com a
moldura seriam duas a mais, e o defeito seria o mesmo que a própria função já
documenta ter sofrido uma vez: caixa que cresce pintando por cima do fluxo.

`InputHeight` passou a ser a altura inteira, moldura incluída, e é ela que o
layout reserva. A guarda que existe exatamente para isso —
`TestTheFrameReservesExactlyWhatTheBoxDraws` — apontou a divergência na primeira
execução.

## Um teste que estava a uma linha de quebrar

`TestADiffNeverRendersTheWholeFile` renderizava um terminal de 40 linhas e
procurava a mudança nele. Isso fazia uma afirmação **sobre o diff** depender de
quantas linhas o layout deixasse: o fluxo é ancorado no fim, a mudança fica no
começo de um bloco de quarenta linhas, e o fixture estava a uma linha de falhar.

As duas réguas gastaram essa linha, e o teste acusou o diff de estar quebrado
quando o diff estava certo. Agora ele afirma sobre o fluxo, que é o que ele diz
verificar.
