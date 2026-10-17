## en

- **The Claude window is declared per model.** The family reported 200K for
  every model, while every model since `claude-opus-4-6` holds 1M, so a
  session was compacted at a fifth of what it could keep. The window now
  comes from the same per-model table as the output ceiling. A model the
  table does not know keeps 200K, because a window that is too large is a
  request refused mid-turn.

## pt-BR

- **A janela Claude é declarada por modelo.** A família devolvia 200K para
  todo modelo, e todo modelo desde `claude-opus-4-6` tem 1M, então a sessão
  era compactada a um quinto do que cabia. A janela agora vem da mesma tabela
  por modelo do teto de saída. Modelo que a tabela não conhece segue com 200K,
  porque janela grande demais é requisição recusada no meio do turno.
