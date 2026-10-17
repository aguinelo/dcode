# A coluna da orquestradora

**Data:** 2026-10-17
**Specs afetadas:** `202608081250-client-tui` (`.p`, §2.2 nova e §10)

Numa sessão com `role: orchestrator`, a metade de cima da coluna lateral é a lista viva dos
filhos em vez do diff (§2.2), e a coluna abre pelo papel, sem esperar arquivo tocado. Onze
invariantes novas na §10, cada uma com teste em `internal/tui/agents_test.go`.

O porquê de cada escolha — a lista como dobra do fio, tokens e não dólares, retenção pela resposta
do daemon, `failed` que fica — está no changelog da `orchestration`,
`202610172330-a-orquestradora-lista-os-agentes.md`, porque é o passo 13 da `.i` dela.

Nenhuma tecla e nenhum comando novos: as tabelas da §7 e da §8 não mudam. `^Q`, `^R` sobre os
filhos e `dcode orchestrate` entram nelas com o resto do passo 13.
