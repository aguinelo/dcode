## en

- **`memory.enabled` and `memory.max_entries` are configurable.** The code
  read both, but neither was a known key, so a `config.toml` setting them was
  refused and `DCODE_MEMORY_ENABLED` / `DCODE_MEMORY_MAX_ENTRIES` were never
  looked up. A new guard checks the other direction too: every key a product
  source reads by name must be in `KnownKeys`.

## pt-BR

- **`memory.enabled` e `memory.max_entries` são configuráveis.** O código lia
  as duas, mas nenhuma era chave conhecida, então um `config.toml` que as
  escrevia era recusado e `DCODE_MEMORY_ENABLED` / `DCODE_MEMORY_MAX_ENTRIES`
  nunca eram consultadas. Uma guarda nova confere também a outra direção: toda
  chave que uma fonte de produto lê pelo nome tem de estar em `KnownKeys`.
