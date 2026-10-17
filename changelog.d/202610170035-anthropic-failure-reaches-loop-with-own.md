## en

- **An Anthropic failure reaches the loop with its own class.** A 400 that says
  the prompt is too long is now `context_size`, which compacts and retries,
  instead of `bad_request`, which aborted the turn on the one failure
  compaction exists to solve. An in-stream `error` frame, such as
  `overloaded_error`, now ends the stream with the class of its HTTP twin and
  the provider's message, instead of being ignored and ending as `transport`
  with the message lost.

## pt-BR

- **Uma falha da Anthropic chega ao laço com a sua própria classe.** Um 400
  que diz que o prompt é longo demais agora é `context_size`, que compacta e
  repete, em vez de `bad_request`, que abortava o turno justo na falha que a
  compactação existe para resolver. Um frame `error` dentro do stream, como
  `overloaded_error`, agora termina o stream com a classe do seu par HTTP e a
  mensagem do provedor, em vez de ser ignorado e terminar como `transport`,
  com a mensagem perdida.
