# ASCII alcança o modal

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** quadro real renderizado em `Unicode: false`

## O que mudou

Oito literais Unicode que nunca entraram nas tabelas de glifos, e a guarda que
os deixava passar.

## A guarda perguntava a coisa errada

`TestNoBoxDrawingRuneSurvivesAsciiMode` pergunta se **um conjunto conhecido** de
glifos escapou. O conjunto é derivado de `glyphs(true)` e `railGlyphs(true)`, o
que é uma melhoria real sobre listá-los — mas só cobre o que as tabelas já
conhecem.

O **modal de aprovação** é desenhado inteiro a partir de literais, e nunca
apareceu num modelo que esse teste montasse. Estava fora da guarda desde o dia
em que ela foi escrita. A tela que pergunta se uma fronteira pode ser cruzada
era a única que um terminal em ASCII não conseguia ler.

A pergunta decisiva é outra: **em ASCII, toda runa é ASCII?** O modelo do novo
teste é montado inteiramente em ASCII, então qualquer runa acima de 127 na saída
veio do layout desenhando. E ele abre o modal.

## Os oito

| Onde | O que vazava |
|---|---|
| `renderApproval` | a moldura inteira: `┌ ┐ └ ┘ ─ │` |
| `renderThought` | a calha `│` do bloco expandido |
| `turnSection` | o `·` de `em vôo 1·4` |
| `assemble` | o `│` que separa os segmentos da barra |
| `diffSegment` | o `−` e o `·` de `+38 −2 · 3 arq` |
| `sessionRow` | o `…` do título cortado e o `·` do nome dado |
| `PlanSummary` | o `·` de `6 de 7 · 1 bloqueado` |
| `summariseResult` | o `−` de `+24 −2` |

## A regra que os dois últimos ensinaram

Os seis primeiros são do renderizador e viraram campos das tabelas: `dot`,
`minus`, `ell` na coluna, e as cinco peças da caixa.

Os dois últimos estavam no **modelo**, que não sabe nada sobre o que o terminal
desenha — e não deve saber. Não ganharam glifo: passaram a produzir ASCII.
`+24 -2`, e `6 de 7 (1 bloqueado)` em vez do ponto médio.

**O modelo produz texto; o glifo é do renderizador.** É a regra que faltava
estar escrita, e a razão pela qual estes dois vazaram por último: eles não eram
o mesmo tipo de defeito que os outros seis, e consertá-los como se fossem teria
levado a geometria para dentro do redutor.

## Por que oito e não cinco

O changelog da coluna lateral já registrou quatro desses achados, um a um, cada
um **depois** do anterior ter sido corrigido. Este é o mesmo padrão pela quinta
vez. A diferença é a forma da afirmação: enquanto a guarda perguntar por um
conjunto, o nono espera pelo nono par de olhos.
