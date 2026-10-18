# Um projeto tem uma orquestradora, que sobrevive ao daemon

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §4, §8, §13, §14; `.config` §1, §2; `.i`,
passo 5).

## O que mudou

O passo 5 da primeira fatia. `orchestrate.Registry` guarda os projetos que o daemon conduz, um por
árvore: `Open(workspace)` resolve a raiz pela `ResolveProject` (a do passo 11, uma só), devolve o
projeto que já existe ou cria um — a orquestradora pela `Spawner`, a linha `project.opened` no log,
e o `state.json` atômico, nessa ordem. `Reload`, no boot, reabre cada projeto do disco: a
orquestradora volta **sob o mesmo id** pelo `Manager.Reopen` do passo 3, o turno que o registro
mostrava aberto termina `daemon_restart`, e `daemon.restart` vai ao log com o que reabriu, o que não
pôde e a linha que `Torn` cortou. As rotas `POST /v1/projects` (201 criado, 200 existente),
`GET /v1/projects`, `GET /v1/projects/{id}` e `GET /v1/projects/{id}/orchestrator` respondem.
`orchestrator.enabled`, `orchestrator.profile` e `orchestrator.main_branch` passam a ser lidas.

## O que o código decide onde a `.p` deixava a forma em aberto

- **`Open` devolve se criou.** A rota responde 201 ou 200, e só quem abriu sabe qual; a assinatura
  da §4 ganhou o `bool`.
- **A `Spawner` cresce com os passos.** Nesta, `SpawnOrchestrator`, `Reopen` e `Close`; `SpawnChild`,
  `SpawnReviewer` e `Submit` entram nos passos que despacham filhos, revisores e relatos. Um método
  sem chamador seria código morto, e `SubmitOrigin` segue com a dispensa do guarda até o passo 7.
- **O `State` também cresce.** `state.json` v1 tem `project` e `log_seq`; os campos de filhos,
  perguntas, fila, saúde da main, teto e relatos entram com os passos que os preenchem — aditivos,
  sem mudar a versão.
- **`enabled` e `dir` são do daemon, `profile` e `main_branch` do projeto.** As duas primeiras são
  lidas onde o `dcode serve` nasce: é o daemon que conduz, e o `Reload` precisa delas antes de
  qualquer pedido. As outras duas, na cadeia da raiz do projeto, quando ele abre; valor inválido é
  `workspace_invalid` com a chave e o valor, ali, porque só ali se sabe a raiz.
- **Um diretório, um daemon.** O `Registry` trava `orchestrator.dir` (`daemon.lock`, `flock`). Dois
  daemons no mesmo diretório reabririam uma orquestradora duas vezes e escreveriam um log com duas
  sequências; o segundo é recusado, dizendo por quê. O daemon embutido num cliente de terminal nunca
  conduz — morre com o cliente, e dois deles seriam esse mesmo defeito — e o `orchestrator_unavailable`
  dele diz que quem conduz é o `dcode serve`.
- **Gravar é condição.** Sessão comum sem registro continua sessão (o aviso de sempre); a
  orquestradora sem registro é fechada e recusada, porque não voltaria de um reinício.
- **A orquestradora não sai do modo `plan`.** `SetMode` recusa outro modo numa sessão
  `RoleOrchestrator`, e o `DELETE` dela é `invalid_input`, como a §5.1 já dizia; sem isso uma troca
  de modo daria a ela a árvore da pessoa, e um `DELETE` deixaria o projeto apontando para nada.
- **O registro da orquestradora não é podado.** `session.Prune` lê o `session.created` de cada
  registro que iria apagar e poupa o de papel `orchestrator`, fora do orçamento. Lido do registro,
  não da lista viva: um daemon embutido, que poda o mesmo diretório e não conduz nada, também o poupa.
- **Projeto no disco que o daemon não tem é reaberto, nunca substituído.** Se o `Reload` falhou ou
  não rodou, o `Open` daquela árvore tenta reabrir; falhando de novo, o erro volta. Um projeto novo
  por cima de um `state.json` ilegível seria uma orquestradora esquecida em silêncio.
- **`Forge` fica vazio** até o passo 10, que decide `github` ou `local`: dizer `local` agora seria
  uma afirmação que ninguém mediu.

## Invariantes

Sobem da §14 para a §13, com teste em `internal/app/orchestrate_test.go`: uma orquestradora por
projeto (`TestTwoOpensOfOneTreeAreOneProject`) e a volta sob o mesmo id, com o histórico
(`TestTheOrchestratorSurvivesADaemonRestart`).
