# A spec segue o código

**Data:** 2026-10-18
**Specs afetadas:** `202608081203-configuration` (`.p`, `.config`)

> **Regra:** correção de documentação. Nenhum comportamento muda; a spec passa
> a dizer o que `internal/config` e `cmd/dcode` já fazem.

## O que estava errado

Uma revisão das specs contra o código encontrou cinco pontos em que esta
família descrevia um produto que não é o que roda:

- **O comando de diagnóstico.** O `.p` (seção 5) e o `.config` (seção 4)
  mandavam usar `dcode config get <chave>`. Não existe subcomando `get`: o
  comando é `dcode config <chave>`, e `dcode --config <chave>` responde o mesmo
  (`cmd/dcode/config.go`, `cmd/dcode/once.go`). Quem seguisse a spec recebia
  `get is not set` e a lista de chaves.
- **O exemplo de `config.toml`.** Mostrava `allow_network = false`, e o default
  é `true` desde a decisão da `sandbox-policy` (`internal/app/app.go`,
  `sandbox-policy.config`). Faltavam chaves que existem e são lidas:
  `model.base_url`, `model.window`, `model.anthropic_workspace`,
  `limits.max_output_tokens`, `memory.enabled`, `memory.max_entries`, e as
  `orchestrator.*`. O exemplo agora traz as que cabem num recorte, diz que é
  recorte, aponta `config.KnownKeys` como o esquema inteiro e a `.config` da
  família dona de cada chave — o orquestrador em
  `orchestration/202610132000-orchestration.config.spec.md`.
- **As assinaturas da seção 5.** `Value.Value` era `any`, e é `string`: a
  camada guarda o texto, e o tipo é decidido por quem lê, pelos acessores.
  `Resolve` devolvia `map[string]Value`, e devolve `Resolved`, com `Warnings`
  (as tentativas de sobrescrever chave travada) e os acessores `Get`, `String`,
  `Bool`, `Int` e `Keys`. `Layer` era citado e não declarado.
- **Tabelas vazias no `.config`.** As seções 2 e 4 tinham cabeçalho de tabela e
  nenhuma linha — o resto das chaves que `202608110900` removeu. A seção 4
  ainda citava um `TRACE` que é `DCODE_CONFIG_TRACE`, uma das removidas. As
  duas seções agora dizem em prosa que não declaram chave, e por quê.
- **O índice de changelog.** O `.p` não listava `202608101900` nem
  `202608110900`; o `.config` listava uma de duas que o tocam.

## O que não mudou

Nenhuma chave, nenhum default, nenhuma invariante. As linhas da seção 7 do `.p`
ficam como estavam, e com elas as reivindicações de teste em
`internal/config/invariants_test.go`.
