# Convenções de `internal/legacy/`

Este diretório é código antigo e tem regra própria, que **substitui** a da raiz
aqui dentro:

- Auxiliares são **não exportados** e usam o prefixo `legacy`.
  Exemplo: `legacyTrimSuffix`.
- O prefixo `Must` não é usado neste diretório em hipótese nenhuma.
