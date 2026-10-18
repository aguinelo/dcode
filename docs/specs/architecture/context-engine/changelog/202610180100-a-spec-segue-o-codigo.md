# A spec segue o código

**Data:** 2026-10-18
**Specs afetadas:** `202608072333-context-engine` (`.p`, `.config`, `.i`)

> **Regra:** correção de documentação. Nenhum comportamento muda; a spec passa
> a dizer o que `internal/contextengine` já faz.

## O que estava errado

- **Uma chave que não existe.** O `.config` mandava ajustar `DCODE_COMPACT_AT`
  e descrevia as faixas do aviso de orçamento movendo-se com ela. Não há chave
  nenhuma: o gatilho é `CompactAt = 0.80`, constante de `DefaultConfig`
  (`internal/contextengine/types.go`). Agora está declarado como constante na
  seção 3, com o motivo, e a prosa da seção 2.1 fala de `CompactAt`. A guarda
  de chaves declaradas não pegou porque o nome só aparecia em prosa, nunca em
  linha de tabela.
- **O nome do pacote.** O `.p` e o `.i` diziam `internal/context`. O pacote é
  `internal/contextengine`, e o nome é de propósito: `context` sombrearia o
  pacote da biblioteca padrão que todo arquivo importa.
- **Tipos que cresceram sem a spec.** `Message` ganhou `Images`, `Documents` e
  `Reminder`; `Image`, `Document` e `Config` não estavam declarados;
  `CompactionPlan` tinha na spec um campo `Keep` que o código nunca teve;
  `Apply` e `Crossed` não apareciam. Tudo isso entra no `.p` como o código é.
- **Documentos anexados.** `InlineDocuments`, a forma exata do marcador
  (atributos entre aspas e escapados em HTML, fronteira de 24 hex derivada do
  SHA-256 do conteúdo, fechamento `</document boundary="B">`), `PDFPages` e o
  custo `charsPerPDFPage = 12000` entram no `.p` (seção 3.1) e na tabela de
  constantes, junto com `KeepTurns = 4`, `KeepFraction = 0.30`,
  `CharsPerToken = 3.5` e `Margin = 0.10`, que estavam só no código.
- **Resumo que falha.** A seção 5 dizia que o loop "devolve `Summary`
  preenchido", sem dizer o que acontece quando o resumo falha. Desde
  `agent-loop/202610161200`, resumo com erro, vazio ou cortado não aplica o
  plano e emite `session.error` `compaction_failed`. A seção 5 e a tabela de
  constantes agora dizem isso e apontam a decisão.
- **O índice de changelog.** O `.p` listava uma das cinco entradas desta
  família; o `.config`, duas.

## O que não mudou

Nenhum valor, nenhuma chave, nenhuma invariante. As linhas da seção 7 do `.p`
ficam como estavam, e com elas as reivindicações em
`internal/contextengine/invariants_test.go`.
