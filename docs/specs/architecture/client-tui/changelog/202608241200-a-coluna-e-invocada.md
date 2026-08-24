# A coluna é invocada, não residente

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** medição sobre a sessão real `1a030dd642cd50b9f68`

## O que mudou

A coluna de arquivos nasce escondida. `^B` a traz, e ela fica como foi deixada.

**É mudança de contrato numa superfície `stable`** — o padrão de uma coluna que
alguém já viu aparecer sozinha. MINOR, no mínimo.

## A medida que decidiu

Reproduzindo a sessão real e renderizando:

| terminal | coluna | painel | **texto** |
|---:|---:|---:|---:|
| 99 | escondida | escondido | **99** |
| 100 | 20 | 25 | **53** |
| 132 | 26 | 33 | **71** |
| 165 | 30 | 34 | **99** |

`RailMinTotalWidth` e `PanelMinTotalWidth` são **os dois 100**. Passar de 99 para
100 colunas custava 46 colunas de leitura de uma vez, e eram necessárias **165
colunas para voltar ao que 99 já dava**.

Quem trabalha entre 100 e 140 — a janela normal — estava no pior ponto da curva:
a tela ocupada e o texto apertado ao mesmo tempo. Alargar o terminal **encolhia**
o texto.

## E o que ocupava as 47%

A coluna repetia o que o fluxo tinha acabado de dizer. Cada `⏺ write DCODE.md`
era seguido, três linhas ao lado, de `✓ DCODE.md`. Vinte e seis colunas é caro
para uma repetição.

O painel, na mesma tela, gastava trinta e três para dizer `iteração 0/2000` e
`em vôo 0·4`.

## O modo automático foi embora

Com o padrão escondido, "deixar a largura decidir" não tinha mais quem o
selecionasse — e o `specguard` disse isso na cara, antes de eu perceber sozinho.

Ele **não** ficou com uma razão escrita. Foi apagado, e com ele a regra de
largura da coluna inteira.

A espelhagem com o painel era o defeito. Os dois respondiam "eu devo estar
aqui?" do mesmo jeito, com o mesmo limiar, e os dois cem **se somaram**. Além
disso guardam coisas diferentes: o painel guarda o plano, que só existe quando o
modelo fez um e é algo a que se volta; a coluna guardava uma segunda cópia do que
o fluxo tinha acabado de dizer. Uma regra que serve ao primeiro não serve à
segunda, e escrever uma regra para os dois foi como ela acabou não servindo a
nenhum.

O teste da regra antiga ficou, **invertido**. Apagá-lo deixaria a regra sem
registro nenhum, e ela é a razão pela qual a coluna agora nasce escondida.

## Por que escondida e não um limiar maior

Layout persistente de múltiplos painéis — lazygit, k9s, btop — funciona porque
**a memória espacial vira navegação**: você aprende que aquilo fica ali e o olho
vai sozinho. Isso pressupõe informação à qual se **volta**. Uma lista que repete
o fluxo não é isso.

Subir o limiar moveria o degrau de lugar sem tirá-lo. Nascer escondida o remove
de vez.

## `^B` continua significando o que significa

No editor de onde a tecla veio, a barra lateral é alternada e **fica como foi
deixada**. Era exatamente esse comportamento que faltava: a tecla estava certa e
o padrão é que discordava dela.

A coluna escondida já dizia que existe e nomeava a tecla. Essa metade estava
pronta e é o que torna o novo padrão descobrível em vez de silencioso.
