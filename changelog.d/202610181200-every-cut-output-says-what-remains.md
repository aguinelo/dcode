## en

- **Every cut output says what remains.** `grep`, `symbol` and `explore`
  marked a cut `Truncated` with `Remaining` at zero, a foreground `bash` cut
  its output without marking it at all, and an echoed `edit` diff cut at the
  cap said so only in the text. Each now fills `Remaining` in the unit of the
  limit that cut it (matches, diff lines, bytes) and says the same number in
  the text; `bash` and `explore` also cut on a whole character. A `bash` that
  hits its deadline is no longer marked `Truncated`: a deadline cuts time, not
  output, and `Code = timeout` already says it. The invariant's test now reads
  every tool in the package, so a new one cannot cut in silence.

## pt-BR

- **Toda saída cortada diz o que ficou de fora.** `grep`, `symbol` e `explore`
  marcavam o corte como `Truncated` com `Remaining` em zero, o `bash` em
  primeiro plano cortava a saída sem marcar nada, e o diff ecoado de `edit`
  cortado no teto só o dizia no texto. Cada um passa a preencher `Remaining`
  na unidade do teto que cortou (ocorrências, linhas de diff, bytes) e a dizer
  o mesmo número no texto; `bash` e `explore` também cortam em caractere
  inteiro. O `bash` que estoura o prazo deixa de ser marcado `Truncated`: o
  prazo corta tempo, não saída, e `Code = timeout` já o diz. O teste da
  invariante passa a ler toda ferramenta do pacote, para que uma nova não
  corte em silêncio.
