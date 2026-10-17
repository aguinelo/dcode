# A proposta anunciada leva as anteriores

**Data:** 2026-10-16
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §5.1). O
detalhe da decisão está em
`done-qualifier/changelog/202610161200-toda-proposta-e-guardada.md`.

## O que mudou

`DoneProposal`, o payload de `done.proposed`, ganha três campos:

| Campo | O que é |
|---|---|
| `sequence` | o número da proposta escrita no turno qualificador, de 1; sempre a última |
| `proposed_at` | quando o modelo a mandou |
| `earlier` | as propostas anteriores do mesmo turno, da mais antiga à mais nova, cada uma com `sequence`, `proposed_at`, `criteria` e `protected`; ausente quando houve uma só |

Os critérios de `earlier` são `ProposedCriterion` — nome, comando, `exit_code`,
`expects` e `why` —, sem `class`, `exit` nem `mismatch`: só a última é medida.
`CommitDoneResponse` ganha `proposals`, quantas o turno fez.

A rota e o evento são os mesmos: `POST /v1/sessions/{id}/done` e
`done.proposed`. Um cliente que não lê os campos novos continua lendo a proposta
escrita como lia.

## Por quê

A `done-qualifier` promete que o operador recebe todas as propostas, em ordem,
como defesa contra uma reproposta que troca um critério por um trivial. Para a
janela e a TUI mostrarem isso, o histórico precisa estar no fio, no aviso que
eles já leem.

`experimental`, abaixo de 1.0: **MINOR**. Campos acrescentados, nenhum removido.
