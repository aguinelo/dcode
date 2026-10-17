# Toda chave lida está no esquema

**Data:** 2026-10-16
**Specs afetadas:** `202608081203-configuration` (`.p`: uma linha nova em §7);
`202608180133-learned-memory` (`.config`: a tabela passa a abrir pela variável,
como as das outras famílias, para que `TestEveryKnownKeyIsDeclaredInSomeSpec`
a leia — nenhuma chave, tipo ou padrão muda)
**Fonte:** defeito registrado como aberto em
`client-server-protocol/changelog/202610080001-a-memoria-e-as-skills-de-um-workspace.md`.

## O defeito

`memory.enabled` e `memory.max_entries` estão declaradas em
`learned-memory.config`, e o `FromEnv` lê as duas — mas nenhuma estava em
`KnownKeys`. O resultado:

- um `config.toml` com `[memory]` era recusado como chave desconhecida;
- `DCODE_MEMORY_ENABLED` e `DCODE_MEMORY_MAX_ENTRIES` nunca eram consultadas,
  porque é `KnownKeys` que diz qual variável carrega cada chave.

A configuração existia só como o seu padrão. Na `main`:

```
a config.toml that sets the memory keys was refused: … Known keys: …
DCODE_MEMORY_ENABLED=false left memory on
DCODE_MEMORY_MAX_ENTRIES=9 gave 40
```

## O que mudou

- As duas chaves entram em `KnownKeys`, com as variáveis e os padrões que
  `learned-memory.config` já declarava (`true`, `40`), e ganham linha na tabela
  de fiação.
- **A guarda passa a olhar nas duas direções.** `TestEveryKnownKeyIsAccountedFor`
  parte de `KnownKeys` e prova que cada chave é lida. Não enxerga o inverso: uma
  chave que o código lê e que nunca entrou no esquema. `TestEveryKeyReadIsAKnownKey`
  percorre as fontes de produto (`internal/`, `cmd/`), coleta toda leitura
  `x.{Bool,String,Int,Get}("seção.chave")` e reprova a que não está em
  `KnownKeys`.

## Por quê

A guarda anterior nasceu de chaves declaradas e lidas por ninguém. Este é o
defeito espelhado — lida e não declarada — e é igualmente silencioso: a leitura
compila, devolve o padrão, e nenhum teste por camada cruza as duas pontas.

## O que NÃO mudou

O comportamento padrão da memória: ligada, teto de 40. Só passa a ser possível
mudá-lo, como a spec da memória sempre disse que era.
