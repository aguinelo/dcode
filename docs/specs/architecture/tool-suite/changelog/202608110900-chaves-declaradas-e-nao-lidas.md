# Chaves declaradas e não lidas saem da spec

**Data:** 2026-08-11

## O que mudou

7 chaves desta spec foram removidas:

- `DCODE_EDIT_REQUIRE_READ`
- `DCODE_GLOB_MAX_RESULTS`
- `DCODE_GREP_MAX_MATCHES`
- `DCODE_READ_MAX_LINES`
- `DCODE_READ_MAX_LINE_LENGTH`
- `DCODE_RESPECT_GITIGNORE`
- `DCODE_WRITE_ATOMIC`

## Por quê

Cada uma estava declarada aqui, era aceita pelo esquema, aparecia em `dcode config` com origem — e **nenhuma linha de código a lia**. Levantamento em 2026-08-11: das 112 chaves declaradas em tabela nas specs de arquitetura, 64 não eram referenciadas em lugar nenhum.

É o mesmo defeito que o `fix/dead-config-keys` corrigiu para quatro chaves, e que `TestEveryKnownKeyIsAccountedFor` passou a impedir dentro de `KnownKeys`. A guarda nunca alcançou o outro lado: uma chave que a spec declara e que nunca chegou a `KnownKeys` não falha teste nenhum, e continua parecendo configuração.

**Superfície de configuração declarada e inexistente é pior que ausente**, porque promete um controle que não existe. Quem escreve a chave no `config.toml` não recebe erro — o valor é lido, resolvido, exibido, e ignorado.

## O que NÃO mudou

**Remover a chave não remove o comportamento.** `GrepMaxMatches` continua em `200` e `ReadMaxLines` em `2000`, fixos em `DefaultLimits()`. O que sai é a promessa de que dá para mudá-lo por configuração, promessa que nunca foi verdadeira.

Se e quando alguma precisar existir, entra de novo — aí com o código que a lê no mesmo PR, que é o que a guarda de `KnownKeys` já exige de qualquer chave nova.

## Precedente

É a mesma decisão que `202608101800` tomou para `DCODE_DOCTRINE_STYLE`: declarada desde a criação da spec, nunca implementada, removida. E a mesma que `202608102000` tomou para `DCODE_VERIFY_TIMEOUT` e `DCODE_VERIFY_FORCE_TURN`, ali por duplicarem outra chave.
