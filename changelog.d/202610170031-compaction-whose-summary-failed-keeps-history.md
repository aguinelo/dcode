## en

- **A compaction whose summary failed keeps the history and says so.** The
  summariser took a stream that ended in an error or at the output ceiling as a
  whole summary, and the loop swapped a failed or empty one for a fixed
  sentence, cut the span and announced `session.compacted` — so a 429 or a
  revoked credential dropped part of the session behind a notice that said it
  was summarised. Now nothing is cut without a real summary, and
  `session.error` `compaction_failed` names the cause: the automatic trigger
  lets the turn go on, `/compact` returns the error, and a `context_size`
  refusal that cannot compact ends the turn and says so.

## pt-BR

- **Compactação cujo resumo falhou mantém o histórico e diz por quê.** O
  resumidor aceitava como resumo inteiro um stream que terminava em erro ou no
  teto de saída, e o loop trocava resumo que falhou ou veio vazio por uma frase
  fixa, cortava o trecho e anunciava `session.compacted` — um 429 ou uma
  credencial revogada apagava parte da sessão atrás de um aviso dizendo que ela
  fora resumida. Agora nada é cortado sem resumo de verdade, e `session.error`
  `compaction_failed` nomeia a causa: o gatilho automático deixa o turno
  seguir, `/compact` devolve o erro, e a recusa `context_size` que não consegue
  compactar encerra o turno e diz isso.
