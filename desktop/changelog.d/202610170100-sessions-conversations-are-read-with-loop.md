## en

- **Sessions and conversations are read with `loop_id`.** The core now runs
  `/loop` and marks the sessions a loop opens (step 4 of
  `docs/proposals/loop.md`); `generated.ts` is regenerated with the loop types,
  and the session and conversation decoders accept the field. Nothing in the
  app uses it yet: starting and following a loop are steps 7 and 8.

## pt-BR

- **Sessões e conversas são lidas com `loop_id`.** O núcleo agora roda o
  `/loop` e marca as sessões que um loop abre (passo 4 de
  `docs/proposals/loop.md`); o `generated.ts` é regenerado com os tipos do
  loop, e os decodificadores de sessão e de conversa aceitam o campo. Nada no
  app o usa ainda: começar e acompanhar um loop são os passos 7 e 8.
