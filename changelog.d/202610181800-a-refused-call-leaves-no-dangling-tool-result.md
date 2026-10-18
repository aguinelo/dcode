## en

- **A call the provider refuses before it runs no longer breaks the next
  request.** A session continuing an orchestrator's conversation reads calls to
  `delegate` and makes one; this session declares no `delegate`, so the call is
  refused as `tool_schema`. The loop fed that back as a tool message with no
  call id and no call above it, and MiniMax-M3 answered the next turn with
  `tool result's tool id() not found`. The refusal now reaches the model as a
  note after its answer, is said on the stream as `session.error`
  (`tool_schema`), and gives the turn a round to correct it. Under it, both
  dialects refuse to send a conversation whose tool results do not pair with
  the calls above them, naming the message, instead of letting the provider
  reject it without saying which.

## pt-BR

- **Chamada que o provider recusa antes de rodar não quebra mais o pedido
  seguinte.** Uma sessão que continua a conversa de um orquestrador lê chamadas
  a `delegate` e faz uma; esta sessão não declara `delegate`, e a chamada é
  recusada como `tool_schema`. O laço devolvia isso como mensagem de ferramenta
  sem id e sem chamada acima, e o MiniMax-M3 respondia o turno seguinte com
  `tool result's tool id() not found`. A recusa agora chega ao modelo como nota
  depois da resposta, é dita no fluxo como `session.error` (`tool_schema`) e dá
  ao turno uma rodada para corrigir. Por baixo, os dois dialetos se recusam a
  mandar conversa cujos resultados de ferramenta não formam par com as chamadas
  acima, nomeando a mensagem, em vez de deixar o provedor recusá-la sem dizer
  qual.
