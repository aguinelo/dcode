## en

- **The Anthropic output ceiling belongs to the model, and an answer cut at it
  says so.** A request with no ceiling configured carried a fixed
  `max_tokens: 8192`, a sixteenth of what most current Claude models accept,
  so long answers stopped mid-sentence and the turn ended as done. The Claude
  family now declares each model's published ceiling (128K on current models,
  32K for one it does not know), `limits.max_output_tokens` /
  `DCODE_MAX_OUTPUT_TOKENS` overrides it on both dialects, and an answer the
  provider cuts (`stop_reason: max_tokens`, `finish_reason: length`) ends the
  turn as `max_tokens` rather than passing for finished.

## pt-BR

- **O teto de saída Anthropic é do modelo, e a resposta cortada nele diz.**
  Requisição sem teto configurado levava um `max_tokens: 8192` fixo, um
  dezesseis avos do que a maioria dos modelos Claude atuais aceita, então
  resposta longa parava no meio da frase e o turno terminava como pronto. A
  família Claude agora declara o teto publicado de cada modelo (128K nos
  atuais, 32K para um que ela não conhece), `limits.max_output_tokens` /
  `DCODE_MAX_OUTPUT_TOKENS` sobrescreve nos dois dialetos, e a resposta que o
  provedor corta (`stop_reason: max_tokens`, `finish_reason: length`)
  encerra o turno em `max_tokens` em vez de passar por terminada.
