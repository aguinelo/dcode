# A janela Claude é por modelo

**Data:** 2026-10-15
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seção 2.1)
**Fonte:** conferência da janela da família `claude` contra a que a Anthropic
publica para cada modelo.

## O que mudou

`Claude.Window(model)` devolvia 200.000 para qualquer modelo. Agora lê a mesma
tabela por prefixo que o teto de saída (`internal/provider/claude_models.go`):

- 1.000.000 — `claude-fable-*`, `claude-mythos-*`, `claude-opus-5*`,
  `claude-opus-4-6` a `4-8`, `claude-sonnet-5*`, `claude-sonnet-4-6`,
  `claude-haiku-5*`;
- 200.000 — as linhas 4.5 e anteriores, `claude-haiku-4-5`, `claude-3-haiku`,
  e todo modelo que a tabela não conhece.

## Por que importa

A janela é de onde saem as faixas do orçamento de contexto e o gatilho da
compactação. Com 200.000 para um modelo de um milhão, a sessão era resumida a
um quinto do que cabia, e cada resumo é contexto que se perde.

## Por que 200.000 para o desconhecido

O erro tem dois lados, e eles não custam igual. Janela pequena demais compacta
cedo, e a sessão segue. Janela grande demais faz o dcode achar que sobra
espaço, e quem descobre que não sobrava é o provedor, recusando a requisição
no meio do turno. `DCODE_WINDOW` continua vencendo a família para quem sabe
mais que a tabela.
