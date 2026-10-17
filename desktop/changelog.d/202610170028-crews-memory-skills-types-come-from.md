## en

- **Crew's memory and skills types come from the core.** Written by hand while
  #432 was open, they are now the ones tygo generates from `internal/protocol`;
  `src/protocol/crew.ts` keeps only the decoder that checks an answer against
  them, so a change in the core's shape fails `check:protocol` instead of
  drifting (N6).

## pt-BR

- **Os tipos de memória e skills da Crew vêm do núcleo.** Escritos à mão
  enquanto o #432 estava aberto, agora são os que o tygo gera a partir de
  `internal/protocol`; `src/protocol/crew.ts` fica só com o decodificador que
  confere uma resposta contra eles, e uma mudança de forma no núcleo reprova o
  `check:protocol` em vez de divergir (N6).
