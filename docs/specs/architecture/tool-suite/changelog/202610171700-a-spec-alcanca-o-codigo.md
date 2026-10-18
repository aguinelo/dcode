# A spec alcança o código

**Data:** 2026-10-17
**Specs afetadas:** `202608072337-tool-suite` (`.r`, RN-1; `.p`, §2, §4.7,
§4.8, §4.10, §4.11, §4.12, §5 e §6; `.config`, §4 e §5)
**Fonte:** revisão das specs contra o código. Só documentação: nenhum
comportamento muda.

## O que estava desatualizado

- **A `.r` conta onze ferramentas e o `.p` detalhava nove.** `process` e
  `fetch` não tinham seção. Ganharam §4.10 e §4.11, escritas a partir de
  `internal/tools/process.go` e `fetch.go`; o `ProcessInput`, que estava
  pendurado em §4.8, mudou para a seção dele.
- **`internal/tools` tem ferramentas de outras famílias.** `remember`,
  `done_propose`, `verdict`, `delegate`, `children`, `steer_child` e
  `close_child` vivem no mesmo pacote e não são desta família. O §4.12 diz de
  quem é cada uma, e a nota da RN-1 diz que a conta das onze é só desta.
- **`explore` sem `owns`.** O §4.7 mostrava o `Delegator` e o `ExploreInput`
  de antes da `delegated-writing`. Agora mostra os dois como estão, e a
  invariante sobre `explore.Declare` diz que `owns` é declarado como escrita.
- **A tabela de códigos de erro.** Listava `is_binary`, que nunca existiu —
  binário em `read` é resultado, não erro — e não listava `bad_input`, que é o
  código de argumento que não decodifica, de URL inválida e de corpo não-texto
  em `fetch`, e de identificador desconhecido em `process`.
- **`Result` sem `Code` e sem `Meta`.**
- **`DCODE_FETCH_ENABLED` e `DCODE_FETCH_MAX_BYTES` estavam entre as
  constantes não configuráveis.** As duas são lidas. Foram para uma seção
  "Rede" da `.config`, com a chave TOML no uso.

## Defeito de código encontrado, não corrigido aqui

A invariante *"Truncamento sempre acompanha `Truncated = true` e
`Remaining > 0`"* não vale para `fetch` nem para `process`: os dois marcam
`Truncated` e deixam `Remaining` em zero. A spec continua dizendo o que a regra
é; o código é que está fora dela.
