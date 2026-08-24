# Uma linha é uma linha

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** sessão real reproduzida pelo redutor — `1a030dd642cd50b9f68`, 1334 eventos

## O que mudou

Dois defeitos da linha de chamada, achados no mesmo lugar: o quadro renderizado
a partir de um log de verdade, em vez de um estado que eu escolhi.

## O quadro se desmontava num `curl`

`targetOf` devolvia o `command` cru. Um comando quebrado em quatro linhas — o
`curl` com contrabarra no fim, que é a forma normal de escrever um — entrava
inteiro num `%-*s`, e a segunda linha começava na coluna zero:

```
⏺ bash   …rtureDate=2026-08-24"
done exit 0  1.6s              │ [^p] hide panel
```

A partir dali a coluna lateral, o divisor e o painel ficavam desalinhados até o
fim da tela.

O achatamento ficou em **`clipStyled`**, e não só em `targetOf`. Toda linha de
toda coluna passa por lá, então é o único ponto que consegue prometer o que o
layout assume em todo lugar. Corrigir só na origem consertaria este campo e
deixaria o próximo em aberto — que é exatamente a forma de defeito que esta
família já pagou duas vezes.

`targetOf` achata também, porque a largura contra a qual o valor é medido é
decidida antes de ele chegar à tela: comando de quatro linhas medido como uma
linha longa elide no lugar errado mesmo quando o quadro sobrevive.

## O teste escrito para pegar isso não pegava

`TestRenderNeverExceedsTheTerminalWidth` divide em `"\n"` **antes** de medir.
Uma linha partida em duas virava duas linhas curtas e passava.

O novo teste afirma a **contagem de linhas** contra a altura do terminal. É a
única forma da afirmação que uma quebra não consegue satisfazer.

E o primeiro teste que escrevi para ele passava sem a correção: o comando longo
que eu escolhi perdia a quebra para a truncagem por acidente. Quem chegava à
tela era o comando **curto**, que a elisão nunca toca. O caso que reproduz é o
curto.

## A elisão guardava a ponta errada

`ellipsis` mantém o fim. Para um caminho está certo — o que identifica um
arquivo é o nome, e os diretórios até ele são o que todo mundo no repositório
tem em comum. Para um comando é o oposto, e a sessão mostrou o custo: quatro
linhas seguidas idênticas,

```
⏺ bash   …null | sort -u | head -40
```

para quatro buscas diferentes. O que as distinguia estava nos primeiros vinte
caracteres.

Quem decide é **o valor, não a ferramenta**. `looksLikePath` já é a resposta
desta família para "isto é um caminho", e perguntar a ela mantém uma definição
só — uma segunda lista de nomes de ferramenta divergiria da primeira, que é o
defeito que a `tree.go` documenta em `namesAFile`. Padrão de `grep` e nome de
filho delegado caem no lado do comando, e está certo: os dois se identificam
pelo começo.
