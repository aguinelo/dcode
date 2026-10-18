# Saída cortada diz quanto ficou de fora

**Data:** 2026-10-18
**Specs afetadas:** `202608072337-tool-suite` (`.p`: as linhas de `process` e
de `fetch` dizem `Remaining`; a invariante de §7 passa a ser reivindicada por
`TestTruncatedOutputAlwaysSaysWhatRemains`)

## O defeito

A invariante diz: truncamento sempre acompanha `Truncated = true` **e**
`Remaining > 0`. `fetch`, `process` e `bash` em segundo plano marcavam
`Truncated` e deixavam `Remaining` em zero. Na `main`, um corpo de 4096 bytes
cortado em 512 devolvia `Remaining=0`, e um log de 1214 bytes numa janela de
200 também. A invariante estava reivindicada por um teste de `symbol` que só
olhava `Truncated`, e por isso nada reprovava.

`Truncated` sozinho diz ao modelo que há mais, e não se é uma linha ou um
megabyte — a diferença entre ler de novo e estreitar a pergunta.

## O que mudou

- `clampTail` devolve quantos bytes deixou de fora, em vez de um booleano;
  `process` e `bash` em segundo plano põem esse número em `Remaining`. A
  primeira linha passa a dizê-lo também: `… <m> earlier bytes dropped, keeping
  the last <k>` — antes dizia o teto, não o que foi mantido.
- `fetch` usa o `Content-Length` quando há um. Sem ele, conta o resto do corpo
  lendo até 16 MiB; acima disso, ou se a leitura falha, o número é um piso e a
  linha diz `at least`. A linha de cabeçalho ganha `<m> more not returned`.

## Por quê contar em vez de estimar

Contar um corpo sem `Content-Length` é baixá-lo. O teto de 16 MiB limita o
custo; abaixo dele o número é exato, acima ele é declarado como piso. Um
`Remaining` inventado seria pior que o zero: pareceria fato.

## O que NÃO mudou

O teto, o que é mantido (o começo em `fetch`, o fim em `process`), e o
`Truncated`.

Ficam de fora, e ainda violam a mesma invariante: `symbol`, a busca por
conteúdo em `search.go` e `explore` marcam `Truncated` sem `Remaining`; o
`bash` em primeiro plano corta a saída em `clamp` sem marcar `Truncated`; e o
`bash` que estoura o prazo marca `Truncated` por tempo, onde não há resto em
bytes para dizer.
