# A lista acha o loop que bloqueia

**Data:** 2026-10-16
**Specs afetadas:** `202608252000-loop-command` (`.p` §8) e
`202608072240-client-server-protocol` (`.p` §2, §6, §9).

## Duas regras para a mesma árvore

A recusa de um segundo loop (`loop_already_running`) já comparava árvores:
um loop vivo recusa outro no mesmo workspace, dentro ou em volta dele, com os
links resolvidos. A lista por workspace (`GET /loops?workspace=`) comparava o
caminho exato, só limpo. As duas regras respondiam à mesma pergunta — "há um
loop nesta árvore?" — de jeitos diferentes.

A TUI pergunta à lista antes de começar um `/loop`, para seguir o que espera
revisão ou dizer qual espera. De uma subpasta do workspace do loop, ou por um
link para ele, a lista não achava nada, a TUI pedia um loop novo, e a pessoa
recebia `could not start the loop: …loop_already_running…` em vez de seguir o
loop que ela mesma tinha mandado revisar.

## A lista usa a regra da recusa

O filtro da lista agora resolve o workspace como o começo resolve
(`loopRoot`) e guarda os loops cuja árvore contém ou está contida na dele
(`overlaps`). O cliente acha o loop que o bloqueia, de qualquer grafia da
mesma árvore. Workspace que não pode ser lido é recusado com
`workspace_invalid`: antes a resposta era uma lista vazia, que se lê como "não
há loop aqui".
