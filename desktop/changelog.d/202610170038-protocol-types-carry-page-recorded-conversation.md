## en

- **The protocol types carry the page of a recorded conversation.**
  `src/protocol/generated.ts` is regenerated for the core's new
  `GET /v1/conversations/{id}/events` (`ConversationEventsResponse`,
  `ConversationEventsLimit`). Nothing in the window reads it yet; that is the
  next change, which opens a finished conversation onto its history instead of
  an empty thread.

## pt-BR

- **Os tipos do protocolo trazem a página de uma conversa gravada.**
  `src/protocol/generated.ts` foi regenerado para a nova rota do núcleo,
  `GET /v1/conversations/{id}/events` (`ConversationEventsResponse`,
  `ConversationEventsLimit`). Nada na janela a lê ainda; isso é a próxima
  mudança, que abre uma conversa terminada no histórico dela em vez de numa
  thread vazia.
