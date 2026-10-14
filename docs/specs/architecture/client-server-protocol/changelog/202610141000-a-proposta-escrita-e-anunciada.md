# A proposta escrita é anunciada

**Data:** 2026-10-14
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §5.1). O
detalhe da decisão está em
`done-qualifier/changelog/202610141000-a-proposta-escrita-e-anunciada.md`.

## O que mudou

A §5.1 ganha a linha de `done.proposed`, que agora tem emissor: o commit da
proposta (`POST /v1/sessions/{id}/done`) o emite no log da sessão
qualificadora depois de medir e escrever o `done.toml`. O payload é o
`DoneProposal` da `done-qualifier` §6.1, com `path` novo e sem `turn_id`,
`round` e `expires_at`.

## Por quê

Um evento que existia no fio sem emissor e sem linha na tabela era promessa
sem dono. Agora tem os dois. É o dado que o desktop precisa para desenhar a
revisão da proposta sem ler o `Summary` em prosa nem tocar o disco.

`experimental`, abaixo de 1.0: **MINOR**. Nenhum cliente dependia da forma
antiga, porque nada a emitia.
