# A recusa do modelo é dita

**Data:** 2026-10-16
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seções 4 e 7),
`202608072335-agent-loop` (`.p`, `StopReason` e invariantes),
`202608072240-client-server-protocol` (`.p`, `turn.completed`),
`202608081250-client-tui` (`.p`, seção 6.9 e invariantes)
**Fonte:** decisão do dono — resposta Claude que termina em `stop_reason:
"refusal"` vira falha explícita e classificada do turno, com motivo próprio,
visível no TUI e no desktop, nunca contada como pronta.

## O que mudou

O decodificador Anthropic só lia `max_tokens` do `stop_reason`. Uma recusa —
HTTP 200, `stop_reason: "refusal"`, às vezes com `stop_details` (`category`,
`explanation`) — terminava como `EventDone` limpo, e a verificação de pronto
tomava a recusa, ou a meia resposta escrita antes dela, pela resposta.

Agora:

- **No adaptador**, `stop_reason: "refusal"` termina o stream em `EventError`
  com a classe nova `refused`, não repetível. A mensagem diz que o modelo
  recusou, com categoria e explicação quando vieram. As chamadas de ferramenta
  pendentes são descartadas — inclusive no `message_stop` que ainda chega
  depois —, e o evento leva o `Usage`, porque a resposta recusada é cobrada.
- **No laço**, `refused` encerra o turno em `StopRefused` (`"refused"`),
  emite `session.error` com código `refused` e a mensagem do provedor, e não
  repete. O texto anterior à recusa fica fora do histórico.
- **No protocolo**, `turn.completed` ganha o motivo `refused`.
- **Nos clientes**, o TUI mostra uma nota nas duas línguas ("the model refused
  to answer" / "o modelo recusou responder") e o desktop mostra "Parou: o
  modelo recusou responder", além da entrada de erro com as palavras do
  provedor.

## Por que uma classe de erro, e não uma marca no `EventDone`

`Truncated` é marca no `EventDone` porque a resposta cortada é real e entra no
histórico. A recusa não é resposta. Como erro, todo consumidor que já para em
erro para nela — o resumidor da compactação, o filho delegado, o laço — sem
precisar conhecer campo novo; como marca no `done`, cada um que não a lesse
tomaria a recusa por resposta terminada, que é exatamente a falha a fechar. E
um erro não libera as chamadas pendentes: uma chamada que o modelo escrevia
quando foi parado não é chamada que ele pediu.

O custo foi um só: o `EventError` passa a poder levar `Usage`, que antes só o
`EventDone` levava. Sem isso, o que a recusa custou sumia da conta do turno e
do teto de tokens.

## Por que um motivo próprio, e não `error`

`error` diz que algo deste lado falhou. Na recusa nada falhou: o modelo
decidiu. Quem lê precisa saber que foi recusa para decidir o que pedir no
lugar, e um `/loop` que lê o motivo não deve confundir as duas coisas. Nunca
repete porque é a única falha que uma nova tentativa garante repetir: o mesmo
pedido ao mesmo modelo.

## O que fica de fora

O fallback do lado do servidor (`fallbacks`, beta) não é pedido aqui: escolher
outro modelo para responder o que um recusou é decisão do dono, não do
adaptador. O dialeto OpenAI (`finish_reason: "content_filter"`) também fica
para depois.
