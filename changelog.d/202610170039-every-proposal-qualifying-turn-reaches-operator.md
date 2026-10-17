## en

- **Every proposal of a qualifying turn reaches the operator.** The daemon
  kept one slot, replaced on each `done_propose`, so the history the
  done-qualifier promises as its defence against a re-proposal that swaps a
  criterion for a trivial one never existed. It now keeps every proposal in
  order, with its number and time; the last is measured and written, and
  `done.proposed` carries the earlier ones as the model sent them. A turn that
  proposes past `MaxProposals` (8) is refused with the reason and nothing is
  written — never the oldest dropped to make room.

## pt-BR

- **Toda proposta de um turno qualificador chega ao operador.** O daemon
  guardava um slot só, trocado a cada `done_propose`, e o histórico que a
  `done-qualifier` promete como defesa contra uma reproposta que troca um
  critério por um trivial nunca existiu. Agora guarda cada proposta em ordem,
  com o número e a hora; a última é medida e escrita, e o `done.proposed` leva
  as anteriores como o modelo as mandou. Um turno que propõe além de
  `MaxProposals` (8) é recusado com o motivo e nada é gravado — nunca a mais
  antiga descartada para abrir espaço.
