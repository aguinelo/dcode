## en

- **Truncated output says how much was left out.** `fetch`, `process` and a
  background `bash` marked a cut result `Truncated` and left `Remaining` at
  zero, against the tool suite's own invariant. They now fill it in bytes:
  `process` and background `bash` with what the tail window dropped, said on
  the first line too, and `fetch` from `Content-Length` or, without one, by
  counting the rest of the body up to 16 MiB ("at least" past that). A model
  told only "cut" cannot tell a missing line from a missing megabyte.

## pt-BR

- **Saída cortada diz quanto ficou de fora.** `fetch`, `process` e `bash` em
  segundo plano marcavam o resultado cortado como `Truncated` e deixavam
  `Remaining` em zero, contra a própria invariante do conjunto de ferramentas.
  Passam a preenchê-lo em bytes: `process` e `bash` em segundo plano com o que
  a janela do fim deixou de fora, dito também na primeira linha, e `fetch` pelo
  `Content-Length` ou, sem ele, contando o resto do corpo até 16 MiB ("at
  least" acima disso). Um modelo que só ouve "cortado" não distingue uma linha
  faltando de um megabyte faltando.
