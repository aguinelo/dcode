## en

- **The desktop no longer reads `done.signed`.** The core removed the event and
  `SignDoneRequest` (step 1 of `docs/proposals/loop.md`, decision Q1 C): the
  signature is the reviewed `done.toml`, and nothing ever emitted either. The
  regenerated `generated.ts` loses both, and the decoder and the
  "Definição de pronto assinada" note go with them; `done.proposed` stays.

## pt-BR

- **O desktop deixa de ler `done.signed`.** O núcleo apagou o evento e o
  `SignDoneRequest` (passo 1 de `docs/proposals/loop.md`, decisão Q1 C): a
  assinatura é o `done.toml` revisado, e nada nunca emitiu nenhum dos dois. O
  `generated.ts` regenerado perde os dois, e o decodificador e a nota
  "Definição de pronto assinada" saem junto; o `done.proposed` fica.
