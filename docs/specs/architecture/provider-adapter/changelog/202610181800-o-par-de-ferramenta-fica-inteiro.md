# O par de ferramenta fica inteiro

**Data:** 2026-10-18
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seção 7),
`202608072335-agent-loop` (`.p`, invariantes),
`202608072240-client-server-protocol` (`.p`, continuar uma conversa)
**Fonte:** teste ao vivo com MiniMax-M3 — uma sessão comum que continuou a
conversa gravada de um orquestrador falhou no turno seguinte com HTTP 400
`tool result's tool id() not found (2013)`. O id vazio entre parênteses era o
sintoma inteiro.

## A causa

Não estava na reconstrução. `Rebuild` já mantinha cada chamada junto do seu
resultado e deixava de fora a chamada sem resultado; o teste novo do registro
de orquestrador confirma isso, com `delegate`, `children` e `ask` no histórico.

Estava no laço. A sessão comum lê, no histórico herdado, chamadas a
`delegate`, e o modelo faz uma. Esta sessão não declara `delegate`, então o
adaptador recusa a chamada antes de ela chegar ao laço (`tool_schema`, RN-8). O
laço devolvia a recusa ao modelo como **mensagem de ferramenta sem id**
(`internal/loop/turn.go`, no `case provider.EventError` do `stream`),
anexada ao histórico durante o stream — antes até da mensagem do assistente, e
sem chamada nenhuma acima dela, porque a chamada recusada nunca entra na
resposta. O pedido seguinte levava um `tool` com `tool_call_id` vazio, e o
MiniMax o recusou. A recusa também não era dita em lugar nenhum: a chamada não
rodou e a tela não disse.

## O que mudou

- **No laço**, a recusa vira nota depois da resposta (papel `user`,
  `Reminder`), que todo provedor aceita no meio da conversa, e é dita no fluxo
  como `session.error` de código `tool_schema`. Numa resposta com chamadas
  válidas, a nota vem depois dos resultados, nunca entre eles e as chamadas.
  Numa resposta cuja única chamada foi recusada, o turno ganha uma rodada para o
  modelo corrigir — o que o comentário já prometia e o código não fazia.
- **No adaptador**, `CheckToolPairs` é o chão sob o laço e a reconstrução: nos
  dois dialetos, conversa com resultado sem id, resultado que não responde a
  uma chamada da mensagem do assistente logo acima, ou chamada sem resposta é
  recusada antes do envio, como `bad_request` que nomeia o índice da mensagem.
  O provedor não nomeava nem a mensagem nem a chamada; repetir o pedido só
  repetiria a recusa.
- **No dialeto OpenAI**, mensagem de ferramenta sem resultado deixa de ser
  enviada, como o dialeto Anthropic já fazia: não tem chamada para nomear.

## Por que a nota e não o par

A recusa podia ter virado um par — a chamada recusada na resposta, com um
resultado de erro. Mas a chamada recusada é exatamente a que não valida: nome
que a sessão não declara, ou argumentos que não são JSON. Pôr isso no histórico
seria mandar ao provedor uma chamada que ele mesmo pode recusar, e o id dela
nem sempre existe. A nota diz o mesmo ao modelo sem inventar nada.

Chamadas antigas a ferramentas que a sessão nova não tem continuam como
foram feitas: ler o passado não exige a ferramenta declarada, e tirar a
chamada deixando o resultado é a metade solta que o provedor recusa.
