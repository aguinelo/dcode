# Chaves declaradas e não lidas saem da spec

**Data:** 2026-08-11

## O que mudou

10 chaves desta spec foram removidas:

- `DCODE_DIFF_CONTEXT_LINES`
- `DCODE_DIFF_MAX_LINES`
- `DCODE_INPUT_QUEUE`
- `DCODE_INPUT_QUEUE_MAX`
- `DCODE_PANEL_ENABLED`
- `DCODE_PANEL_MIN_TOTAL_WIDTH`
- `DCODE_PANEL_WIDTH`
- `DCODE_STREAM_MAX_LINES`
- `DCODE_TOOL_EXPAND_ERRORS`
- `DCODE_UNICODE`

## Por quê

Cada uma estava declarada aqui, era aceita pelo esquema, aparecia em `dcode config` com origem — e **nenhuma linha de código a lia**. Levantamento em 2026-08-11: das 112 chaves declaradas em tabela nas specs de arquitetura, 64 não eram referenciadas em lugar nenhum.

É o mesmo defeito que o `fix/dead-config-keys` corrigiu para quatro chaves, e que `TestEveryKnownKeyIsAccountedFor` passou a impedir dentro de `KnownKeys`. A guarda nunca alcançou o outro lado: uma chave que a spec declara e que nunca chegou a `KnownKeys` não falha teste nenhum, e continua parecendo configuração.

**Superfície de configuração declarada e inexistente é pior que ausente**, porque promete um controle que não existe. Quem escreve a chave no `config.toml` não recebe erro — o valor é lido, resolvido, exibido, e ignorado.

## O que NÃO mudou

**Remover a chave não remove o comportamento.** A largura do painel continua em `24` e o teto de diff em `40`, fixos em `DefaultGeometry()`. O que sai é a promessa de que dá para mudá-lo por configuração, promessa que nunca foi verdadeira.

Se e quando alguma precisar existir, entra de novo — aí com o código que a lê no mesmo PR, que é o que a guarda de `KnownKeys` já exige de qualquer chave nova.

## Precedente

É a mesma decisão que `202608101800` tomou para `DCODE_DOCTRINE_STYLE`: declarada desde a criação da spec, nunca implementada, removida. E a mesma que `202608102000` tomou para `DCODE_VERIFY_TIMEOUT` e `DCODE_VERIFY_FORCE_TURN`, ali por duplicarem outra chave.
