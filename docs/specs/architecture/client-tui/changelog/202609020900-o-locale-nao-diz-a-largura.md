# O locale não diz a largura de um glifo

**Data:** 2026-09-02
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7.3 e 10)
**Fonte:** medição feita ao avaliar trocar `⏺` por `●` — o glifo do Claude Code
mede duas células em locale asiático, e a pergunta cosmética descobriu um
defeito

## O que mudou

Este pacote mede com régua própria, `ruler`, fixada em uma célula para os
caracteres de largura **ambígua**. Antes media com o global do `go-runewidth`,
que é decidido pelo `init()` da dependência a partir do locale do processo.

## O defeito

Com `LANG=ja_JP.UTF-8`, a tela inteira saía com **o dobro da largura do
terminal**.

Uma variável de ambiente decidia duas coisas que precisam concordar, e as
levava à contradição:

- `supportsUnicode` lê o locale, vê UTF-8 e escolhe **o conjunto de caixa** —
  `│`, `└`, `─`, `·`, `▌`, `…`.
- o `init()` do `go-runewidth` lê o mesmo locale, vê `ja_JP` e passa a medir
  **esse mesmo conjunto** com duas células.

Todo caractere de caixa é ambíguo na tabela East Asian Width do Unicode. Então o
cliente escolhia os glifos exatamente pelo sinal que dobrava a medida deles.

Reproduzido antes de qualquer correção, com os guardas que o produto já tem:

```
RUNEWIDTH_EASTASIAN=1 go test ./internal/tui
  TestRenderNeverExceedsTheTerminalWidth: width 80 → uma linha de 160 células
  TestEveryLineReachesTheRightEdge:       a moldura da entrada, 160 em 80
  TestTheApprovalBlockShows…:             o modal estourou, 200 células
  … onze testes, a caixa de entrada, o modal, a trilha e a marca
```

Onze guardas reprovaram de uma vez. Nenhum deles estava errado: todos rodavam
num processo cujo locale não era asiático, e **a suíte inteira era a mesma
aposta que o produto**.

## O que o locale diz, e o que não diz

Diz qual língua a pessoa lê. **Não** diz quantas células o terminal dela dá a
uma régua vertical, e nenhum sinal portátil diz.

Este produto já faz essa distinção para cor: `NO_COLOR` e `TERM=dumb` são
inferência, `DCODE_COLOR` é a pessoa respondendo pelo próprio terminal. A
diferença aqui é que **ninguém neste repositório escolheu**: quem escolheu foi o
`init()` de uma dependência, a partir de uma variável lida para outra coisa.

## Uma célula, e a saída para quem precisa de outra

Os glifos são medidos como foram desenhados: uma célula. Não porque todo
terminal os desenhe assim, mas porque **este layout é construído sobre eles
nessa largura**, e terminal que os desenha mais largos precisa de um conjunto de
glifos diferente, não de uma aritmética diferente.

Esse conjunto já existe e está a uma variável de distância: `DCODE_ASCII=1`, em
que toda marca tem uma célula por construção. É saída que funciona, ao contrário
de honrar `RUNEWIDTH_EASTASIAN=1` — o layout não desenha corretamente com marcas
de duas células, e dizer que honra seria prometer o que não se cumpre.

## Régua do pacote, não o global reescrito

Reescrever o global daqui alcançaria todo pacote que mede, e o risco passaria a
ser a **ordem em que dois `init()` rodaram**. A régua é do pacote, pedida
explicitamente em cada medida, e imune ao que o locale do processo disser.

## Dois guardas, e o segundo é o que importa

O primeiro liga o global hostil e afirma que a régua não se move, e que a tela
cabe em cinco larguras.

O segundo pergunta ao **código-fonte** se alguém mede por fora da régua. É ele
que previne a volta: o comportamento sozinho continuaria passando com metade das
chamadas no global, porque a falha só aparece num locale em que a suíte não
roda — que é exatamente por onde isto entrou.

**Os testes também medem pela régua do pacote.** Guarda que mede com régua
diferente da que o produto desenha é guarda que reprova o que ninguém vê e
aprova o que todo mundo vê.

## A lista de ambíguos é medida, não lembrada

Escrita de memória, ela veio com `◦`, que é estreito em qualquer locale. O
guarda tem um `Fatal` afirmando que cada caractere da lista mede dois pelo
global — sem ele, um caractere que não é ambíguo passa a ser um caso que o teste
acha que está cobrindo e não está.

Nem toda marca é afetada: `⏺`, `✻`, `⊘`, `✓`, `❯`, `╎`, `▸`, `▾` e `◦` medem uma
célula em qualquer locale.

## O que isso responde sobre a pergunta que o encontrou

A pergunta era trocar `⏺` (U+23FA) por `●` (U+25CF), o marcador do Claude Code.
A medida responde: `⏺` é inequívoco, `●` é ambíguo. O marcador atual é o mais
seguro dos dois, e a semelhança que a troca compraria custaria a única coisa que
uma marca não pode custar — a coluna em que ela começa.

Fica como está, e agora por um motivo medido em vez de por inércia.
