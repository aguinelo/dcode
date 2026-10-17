## en

- **A refusal is said, and never counted as done.** A Claude answer that ends
  with `stop_reason: "refusal"` used to decode as a clean finish, so the turn
  took the refusal, or the half answer streamed before it, for the answer. It
  now ends the stream as a `refused` provider error that is never retried,
  drops the tool calls still arriving and keeps the billed usage. The turn ends
  with the new stop reason `refused`, after a `session.error` with code
  `refused` and the provider's words, and the TUI says the model refused in
  both languages.

## pt-BR

- **A recusa é dita, e nunca contada como pronta.** Uma resposta Claude que
  termina em `stop_reason: "refusal"` era lida como término limpo, e o turno
  tomava a recusa, ou a meia resposta enviada antes dela, pela resposta. Agora
  ela termina o stream como erro de provedor `refused`, que nunca é repetido,
  descarta as chamadas de ferramenta ainda chegando e guarda o uso cobrado. O
  turno termina com o motivo novo `refused`, depois de um `session.error` de
  código `refused` com as palavras do provedor, e o TUI diz que o modelo
  recusou nas duas línguas.
