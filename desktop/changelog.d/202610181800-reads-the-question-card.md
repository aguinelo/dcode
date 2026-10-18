## en

- **The window reads the orchestrator's question cards.** The core now groups
  the questions of one kind that wait together into a card, with a header, a
  description for each option, the orchestrator's summary and, on a merge, the
  evidence read from the decision log; it sends `question.card` and answers a
  whole card in one POST. The desktop regenerates its protocol types and checks
  the new event and the new fields at the boundary, so a card is read and
  never let through unread. Drawing the card in the conversation comes in the
  next change; the agent list's queue works as before.

## pt-BR

- **A janela lê os cartões de pergunta da orquestradora.** O núcleo agora junta
  num cartão as perguntas do mesmo tipo que esperam juntas, com etiqueta,
  descrição de cada opção, o resumo da orquestradora e, num merge, a evidência
  lida do log de decisões; manda `question.card` e responde um cartão inteiro
  num POST só. O desktop regenera os tipos do protocolo e confere o evento e os
  campos novos na fronteira, então um cartão é lido e nunca passa sem leitura.
  Desenhar o cartão na conversa vem na mudança seguinte; a fila da lista de
  agentes funciona como antes.
