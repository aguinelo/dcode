## en

- **A turn that goes round again speaks in a new message.** When the done check
  sends a turn back to work, the model's next words came out glued to its last
  ones — "…o parser.O teste…" — because streamed text joined any open message of
  the same turn. The daemon reports each new round with a progress event; text
  now joins only the message of the same turn and the same round.

## pt-BR

- **Um turno que dá outra volta fala numa mensagem nova.** Quando a checagem de
  pronto manda um turno de volta ao trabalho, as palavras seguintes do modelo
  saíam coladas às últimas — "…o parser.O teste…" —, porque o texto que chega em
  partes se juntava a qualquer mensagem aberta do mesmo turno. O daemon avisa cada
  rodada nova com um evento de progresso; agora o texto só se junta à mensagem do
  mesmo turno e da mesma rodada.
