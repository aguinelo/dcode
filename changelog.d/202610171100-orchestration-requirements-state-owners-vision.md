## en

- **The orchestration requirements state the owner's vision.** The approved
  research of #461 had a conversation split one request into children and
  gather their reports; the owner wants dcode to be the project's orchestrator:
  one conversation per project, alive in the daemon across restarts, answering
  at once while children work in their own worktrees, with three autonomy
  levels, an always-ask list the harness enforces, a gate the orchestrator runs
  itself on the commit it merges, a serial merge queue, declared deploys, a
  live agent list and a decision log. Rewritten because running the workflow by
  hand on 2026-10-16 showed each failure these rules now prevent. The owner
  decided the open questions — the daemon becomes a launchd user agent, each
  child is a PR squash-merged against the measured commit, a child's approval
  always waits for the person, and design questions are asked or recorded per
  autonomy level — and approved the requirements; nothing is built yet.

## pt-BR

- **Os requisitos de orquestração dizem a visão do dono.** A pesquisa aprovada
  na #461 tinha uma conversa que repartia um pedido em filhos e juntava os
  relatos; o dono quer o dcode como orquestrador do projeto: uma conversa por
  projeto, viva no daemon através de reinícios, respondendo na hora enquanto os
  filhos trabalham em worktrees próprias, com três níveis de autonomia, uma
  lista do que sempre pergunta imposta pelo harness, um gate que a
  orquestradora roda ela mesma no commit que faz merge, fila serial de merge,
  deploy declarado, lista viva de agentes e log de decisões. Reescrito porque
  fazer o fluxo à mão em 2026-10-16 mostrou cada falha que estas regras agora
  evitam. O dono decidiu as perguntas abertas — o daemon vira LaunchAgent do
  `launchd`, cada filho é um PR com squash-merge contra o commit medido, a
  aprovação de filho sempre espera a pessoa, e a decisão de desenho é
  perguntada ou registrada conforme o nível — e aprovou os requisitos; nada
  construído ainda.
