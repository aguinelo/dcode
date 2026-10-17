## en

- **The signature round trip that never shipped is removed.** `done.signed`,
  `SignDoneRequest`, `qualifier.Sign` (with `Asker`, `SignedAnswer`,
  `ErrRefused` and the round ceiling) and the `DCODE_QUALIFIER_MAX_SIGN_ROUNDS`
  and `DCODE_QUALIFIER_SIGN_TIMEOUT` keys leave the protocol, the code and the
  spec, with the nine invariants only `Sign` kept. The signature became the
  reviewed `done.toml` when the measurement moved out of the turn, and these
  were left behind with no emitter, no route and no reader but tests and a
  desktop decoder: a promise on a versioned wire that no code kept. A removed
  contract, so MINOR. Decision Q1 C of `desktop/docs/proposals/loop.md`;
  `done.proposed` stays and is rewired as a notice next.

## pt-BR

- **O ida-e-volta da assinatura, que nunca foi entregue, sai.** `done.signed`,
  `SignDoneRequest`, `qualifier.Sign` (com `Asker`, `SignedAnswer`,
  `ErrRefused` e o teto de rodadas) e as chaves
  `DCODE_QUALIFIER_MAX_SIGN_ROUNDS` e `DCODE_QUALIFIER_SIGN_TIMEOUT` saem do
  protocolo, do código e da spec, com as nove invariantes que só o `Sign`
  cumpria. A assinatura virou o `done.toml` revisado quando a medição saiu do
  turno, e estas peças ficaram para trás sem emissor, sem rota e sem leitor além
  de testes e de um decodificador no desktop: promessa num fio versionado que
  nenhum código cumpria. Contrato removido, então MINOR. Decisão Q1 C de
  `desktop/docs/proposals/loop.md`; o `done.proposed` fica e é religado como
  aviso no passo seguinte.
