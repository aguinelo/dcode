# As chaves de memória chegam à sessão

**Data:** 2026-10-08
**Specs afetadas:** `202608081203-configuration` (`.p`: uma linha nova em §7).
A tabela de `202608180133-learned-memory.config.spec.md` já declarava as duas
chaves e não muda.
**Fonte:** deixado de fora de `202610080001-a-memoria-e-as-skills-de-um-workspace`
(client-server-protocol), com branch própria.

## O defeito

`memory.enabled` e `memory.max_entries` eram lidas por `fromResolved` e não
estavam em `KnownKeys`. Nenhuma camada além do padrão chegava a elas:

- um `config.toml`, do usuário ou do projeto, que as escrevesse era recusado
  como chave desconhecida;
- a camada de ambiente só consulta as variáveis que `KnownKeys` mapeia, então
  `DCODE_MEMORY_ENABLED` e `DCODE_MEMORY_MAX_ENTRIES`, declaradas na spec da
  memória, nunca eram lidas.

A memória desligada e o teto só existiam como valor padrão. Na `main`:

```
a config.toml setting the memory keys is refused: …/.dcode/config.toml:2:
unknown key "memory.enabled".
```

## Por que nenhuma guarda viu

Duas guardas olhavam para o lugar certo e não enxergavam.

- **A da fiação parte de `KnownKeys`** e pergunta se cada chave é lida. Nunca
  perguntou o contrário: se cada chave lida está no esquema.
- **A das specs lê só linha de tabela que abre com a variável.** A tabela da
  memória abre com a chave TOML e põe a variável na segunda coluna, então as
  duas guardas de declaração liam aquela spec como se não declarasse nada.

## O que mudou

- **As duas chaves entram em `KnownKeys`**, com as variáveis que a spec da
  memória já declarava, e na tabela de fiação.
- **Os padrões entram na camada `built-in`** (`true` e `memory.DefaultMax`), para
  que `dcode config` responda com valor e origem, como as outras chaves que
  governam comportamento.
- **Guarda nova, `TestFromEnvReadsOnlyKnownKeys`:** toda chave que `FromEnv` lê
  por acessor está em `KnownKeys`.
- **A linha de spec pode abrir com a chave TOML** antes da variável. Com isso a
  guarda das specs passa a ver a tabela da memória, e teria acusado este
  defeito no dia em que a tabela entrou.

## A invariante

A linha nova em §7 é reivindicada por `TestFromEnvReadsOnlyKnownKeys`. O
comportamento é provado por `TestTheMemoryKeysReachASession`, pelo arquivo do
projeto e pelo ambiente.

Correção de comportamento, sem contrato mudado: PATCH.
