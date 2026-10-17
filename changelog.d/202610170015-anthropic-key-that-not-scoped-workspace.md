## en

- **An Anthropic key that is not scoped to a workspace names one.** New key
  `model.anthropic_workspace` / `DCODE_ANTHROPIC_WORKSPACE`, sent as the
  `anthropic-workspace-id` header on the anthropic dialect only, shown by
  `dcode config` and settable per profile in `models.toml`. Such a key had
  every request refused with a message about a header nobody can set in dcode;
  that refusal now names the key that fixes it. The name avoids `workspace`,
  which already means the folder a session runs in.

## pt-BR

- **Uma chave Anthropic sem escopo de workspace nomeia um.** Chave nova,
  `model.anthropic_workspace` / `DCODE_ANTHROPIC_WORKSPACE`, mandada no
  cabeçalho `anthropic-workspace-id` só no dialeto anthropic, mostrada por
  `dcode config` e configurável por perfil em `models.toml`. Uma chave assim
  tinha toda requisição recusada com uma mensagem sobre um cabeçalho que
  ninguém configura no dcode; a recusa agora nomeia a chave que resolve. O
  nome evita `workspace`, que já é a pasta em que a sessão roda.
