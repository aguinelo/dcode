# Chamada com corpo é um bloco, e o card do design já existia

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`)
**Fonte:** `refs/design/HANDOFF.md` (v5, §3)

## O que mudou

Chamada de ferramenta que carrega corpo — diff, ou saída expandida — passa a ter
uma linha em branco separando-a do que está em volta. Chamada sem corpo continua
uma linha só.

## Por que é tão pouco

Porque quase tudo do §3 já estava construído, e descobrir isso foi o trabalho:

| O design pede | Onde já estava |
|---|---|
| `…` enquanto roda, quando não há contagem | `renderToolLine`, `case e.Running` |
| duração só a partir de 500 ms | `renderToolLine`, com o motivo escrito no lugar |
| `240 lines` · `created, 38 lines` · `+24 −2` · matches em arquivos · `exit 0` | `summariseResult`, a coluna "meta pronta" inteira |
| a moldura agrupando corpo e cabeçalho | `detailLines`, que já desenha `│` à esquerda de toda linha de corpo |

O card do design **já existe em unidade de terminal**: a calha `│` é a espinha
que amarra o corpo ao cabeçalho. O que faltava era o respiro em volta — o "gap de
8px entre blocos do fluxo (≈ 1 linha em branco)" que o próprio handoff pede.

## Por que não a borda de runas

Custaria duas colunas e duas linhas por chamada, exigiria variante ASCII, e a
borda entraria na seleção do modo de cópia — que a spec trata como superfície.
E não faria nada que a calha já não faça.

Fica registrada no `docs/ROADMAP.md` como preferência visual com o preço nomeado,
e não como mudança de ideia esperando acontecer.

## O gap vai antes, nunca depois — e isso foi medido

A primeira versão punha uma linha em branco **depois** do corpo. O
`TestADiffNeverRendersTheWholeFile` reprovou: a janela do fluxo é ancorada no
fim, então o branco final empurrou o topo para fora e **a linha alterada do diff
saiu da tela** num terminal de 40 linhas.

Branco no fim custa uma linha do que aconteceu, para não mostrar nada. Então a
separação vem antes do bloco e antes de quem vem depois dele, e o fluxo nunca
termina em branco.

## Nunca duas

Bloco seguido de bloco abriria com o branco que o anterior fechou. Espaçamento
duplo se lê como coisa faltando, não como separação.
