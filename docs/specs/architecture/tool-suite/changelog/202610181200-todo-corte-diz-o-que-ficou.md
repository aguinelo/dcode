# Todo corte diz o que ficou de fora

**Data:** 2026-10-18
**Specs afetadas:** `202608072337-tool-suite` (`.p`: `Remaining` ganha unidade
por ferramenta, numa tabela em §2; as linhas de `grep`, `symbol`, `explore` e
`bash` dizem o que contam; o timeout de `bash` deixa de ser truncamento; a
invariante de §7 cobre toda ferramenta do pacote)

## O defeito

O `202610180320` consertou `fetch`, `process` e `bash` em segundo plano e
deixou registrado o que ainda violava a mesma invariante. Conferido um a um na
`main`:

- `grep` e `symbol` paravam no teto, marcavam `Truncated` e deixavam
  `Remaining` em zero: `stopped at 5 matches` com 115 ocorrências depois.
- `explore` marcava `Truncated` com `Remaining` zero; a interface `Delegator`
  só devolvia um booleano, e o tamanho do corte nem chegava à ferramenta.
- `bash` em primeiro plano cortava em `clamp` sem marcar `Truncated`: 10 000
  bytes viravam 100 e o resultado dizia que estava inteiro.
- `edit` com diff ecoado acima de `DiffMaxLines` dizia `diff truncated` no
  texto e nada no `Result` — não estava na lista, e foi o teste novo que o
  achou.
- `bash` que estoura o prazo marcava `Truncated` sem nenhum byte retido.

## O que mudou

- `Remaining` está na unidade do teto que cortou, uma por ferramenta, e a
  tabela de §2 diz qual. A spec dizia "bytes", e `read` (linhas) e `glob`
  (caminhos) já faziam outra coisa: a regra escrita era a errada.
- `grep` e `symbol` continuam contando depois do teto, sem mostrar. O custo é
  o de uma busca sem ocorrência nenhuma, que já lê a árvore inteira.
- `Delegator.Explore` devolve `dropped int` em vez de `truncated bool`; o laço
  conta os bytes do relatório cortado, e corta em caractere inteiro.
- `clamp` devolve quantos bytes deixou de fora, também em caractere inteiro, e
  o `bash` em primeiro plano marca `Truncated` com esse número.
- `edit` ecoando diff cortado marca `Truncated` com as linhas de diff que
  ficaram de fora e diz o número numa linha própria. O marcador `⋯ diff
  truncated at N lines` ficou como estava porque viaja em `Meta.Diff` e o
  desktop o lê.
- `TestTruncatedOutputAlwaysSaysWhatRemains` lê toda `Execute` do pacote pelo
  código-fonte. Cada uma está entre as que cortam, com cenário e número exato,
  ou entre as que não cortam, com o motivo. Uma ferramenta nova fora das duas
  reprova.

## Por quê o prazo não é truncamento

Truncamento é afirmação sobre bytes: houve saída e parte dela não veio. O
prazo corta o comando, não a saída — tudo o que ele escreveu está no
resultado. Pôr segundos em `Remaining` misturaria unidades no mesmo campo, e
marcar `Truncated` sem `Remaining` era justamente a violação. Quem diz o corte
de tempo é `Code = timeout` com `IsError`, que já existia. Se a saída também
passou do teto, aí há truncamento, contado em bytes como em qualquer outro
`bash`.

## O que NÃO mudou

Os tetos, e o que é mantido: o começo em `bash` de primeiro plano e em
`fetch`, o fim em `process`. Linha longa em `read` continua cortada no lugar,
com `(<n> more bytes on this line)`; não marca `Truncated` porque toda linha
pedida veio, e `Remaining` conta linhas.
