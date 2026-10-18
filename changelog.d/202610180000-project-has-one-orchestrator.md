## en

- **A project has one orchestrator that survives the daemon.** `POST /v1/projects` opens the
  project of a workspace — the root of its main worktree, so every worktree and link of one tree is
  the same project — answering 201 when it creates it and 200 when it finds it; `GET /v1/projects`,
  `GET /v1/projects/{id}` and `GET /v1/projects/{id}/orchestrator` answer too. Each project has one
  orchestrator: a recorded session in plan mode at the root, which refuses to leave plan mode and
  to be deleted, since it never writes to the tree and the project names it. Its state is kept in
  `state.json`, rewritten atomically, and when `dcode serve` starts again every project reopens its
  orchestrator under the same id, with its history; the turn it was in ends `daemon_restart`, and
  `daemon.restart` in the decision log says what reopened, what did not, and what a crash cut. A
  project whose state does not read stays closed and says why — never an empty project in its
  place. Pruning old session records spares an orchestrator's. `orchestrator.enabled` (default
  `false`), `orchestrator.profile` and `orchestrator.main_branch` are now read; with the
  orchestrator off, or sessions not recorded, the routes answer `orchestrator_unavailable` saying
  which, and a daemon embedded in one terminal client says the shared daemon is the one that
  conducts. One daemon holds an `orchestrator.dir`: a second is refused, since it would reopen the
  same orchestrator twice. This is step 5 of the orchestrator's first slice.

## pt-BR

- **Um projeto tem uma orquestradora, que sobrevive ao daemon.** `POST /v1/projects` abre o projeto
  de um workspace — a raiz da worktree principal, então toda worktree e todo link da mesma árvore
  são o mesmo projeto —, respondendo 201 quando cria e 200 quando acha; `GET /v1/projects`,
  `GET /v1/projects/{id}` e `GET /v1/projects/{id}/orchestrator` também respondem. Cada projeto tem
  uma orquestradora: uma sessão gravada em modo `plan` na raiz, que recusa sair do `plan` e ser
  apagada, porque nunca escreve na árvore e o projeto aponta para ela. O estado fica em
  `state.json`, reescrito atômico, e quando o `dcode serve` volta cada projeto reabre a orquestradora
  sob o mesmo id, com o histórico; o turno em que ela estava termina `daemon_restart`, e
  `daemon.restart` no log de decisões diz o que reabriu, o que não pôde e o que uma queda cortou.
  Projeto cujo estado não se lê fica fechado e diz por quê — nunca um projeto vazio no lugar. A poda
  dos registros antigos de sessão poupa o da orquestradora. `orchestrator.enabled` (padrão `false`),
  `orchestrator.profile` e `orchestrator.main_branch` passam a ser lidas; com a orquestradora
  desligada, ou sem gravação de sessões, as rotas respondem `orchestrator_unavailable` dizendo qual,
  e o daemon embutido num cliente de terminal diz que quem conduz é o daemon compartilhado. Um daemon
  só segura um `orchestrator.dir`: o segundo é recusado, porque reabriria a mesma orquestradora duas
  vezes. É o passo 5 da primeira fatia do orquestrador.
