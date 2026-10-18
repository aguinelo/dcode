# Implementing: Orquestrador — primeira fatia

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.md`). Cada passo é
> um PR com um tema só, título sem "and", fragmento em `changelog.d/` e changelog de família; os
> do núcleo são MINOR, os do desktop entram em `desktop/changelog.d/` com D-número.

## Onde isto está, hoje

| Passo | Estado |
|---|---|
| 0 — changelog em fragmentos | **feito** (#515) |
| 1 — `.p`, `.config`, `.i` desta fatia | **feito** (#516) — aprovado pelo dono em 2026-10-17, com as seis decisões do fim |
| 2 — uma sessão pode ter mãe | **feito**, menos `TurnStarted.Origin`, `OriginChildren` e `StopDaemonRestart`, que entram com o passo 3 |
| 3 — a sessão gravada reabre sob o próprio id | **feito** (#519), com os três itens do fio que o passo 2 deixou |
| 4 — toda decisão vai ao log | **feito** — `internal/orchestrate/decisions.go`, e `orchestrator.dir` lida por `internal/orchestrate/dir.go` |
| 5 — um projeto tem uma orquestradora | **feito** — `internal/orchestrate/registry.go`, `store.go`; `app.Daemon` é a `Spawner` com `SpawnOrchestrator`, `Reopen` e `Close` (os de filho entraram no passo 6; o de revisor entra no 8) |
| 6 — um filho trabalha na própria worktree | **feito** — `internal/orchestrate/dispatch.go`, `children.go`, `observe.go`, `owns.go`, `worktree.go`, `limits.go`; as ferramentas em `internal/tools/orchestrate.go` e a negação de git em `gitwrite.go`; `app.Daemon` ganha `SpawnChild`, `Submit` e `Steer`. O `done` deixa o filho em `working` com o fim dito: a esteira (`in_review`) começa ali, no passo 8 |
| 7 — o relato de um filho começa o turno | **feito** — `internal/orchestrate/reports.go`: a caixa (`State.PendingReports`), o debounce de `report_debounce`, o turno `Origin: children` |
| 8 — um revisor lê o diff de cada filho | **feito** — o componente (`review.go`, `review_verdict.go`, `internal/tools/verdict.go`) e a fiação: `internal/orchestrate/pipeline.go` (conferência de `owns` antes do commit, `Spawner.SpawnReviewer`, `missing` uma vez de novo, `max_returns`), `pipeline_gate.go` (fase `red`/`fix`, `red_test.confirmed`) e a ferramenta `internal/tools/redtest.go` |
| 9 — a orquestradora roda o gate | **feito**, menos o sandbox do `bash` em volta do gate e a guarda de fonte do `vcs`: a esteira atualiza a branch na integração (`Integration.Update`) e mede `H` (`pipeline_gate.go`); o conflito volta ao filho, e o turno seguinte conclui o merge |
| 10 — um merge espera a pessoa | **feito** — o componente (`merge.go`, `integrate.go`, `forge.go`, `forge_local.go`) e a fiação em `internal/orchestrate/merging.go` (`Propose` no verde, `Answer`, a fila serial, `merged`), `questions.go` e a ferramenta `internal/tools/ask.go`; perguntas, fila, `Main` e `LocalSaid` no `State`; as rotas de `/questions` em `internal/server/orchestrate.go` |
| 11 — `dcode decisions` lê o log | **feito** — `cmd/dcode/decisions.go`, com `LogReader` e `ResolveProject` em `internal/orchestrate/` |
| 12 | **não feito** |
| 13 — a orquestradora lista os agentes | **feito** — a lista na coluna lateral, dobrada do fio (`internal/tui/agents.go`, `agents_side.go`); `dcode orchestrate` (`cmd/dcode/orchestrate.go`), a leitura de `/children` e `/questions` ao abrir, `^Q` e `^R` sobre os filhos (`internal/tui/orchestrate.go`, `orchestrate_view.go`) |
| 14 | **não feito** |

## Pré-requisitos

| Componente | Estado mínimo |
|---|---|
| `202610132000-orchestration.r` | aprovado (#514) |
| `202608191800-delegated-writing` | `owns`, `Resolver.Owning`, `childRegistry` — `internal/loop/delegate.go` |
| `202608072240-client-server-protocol` | sessões, SSE por sessão, aprovações com `ExpiresAt` |
| `202608261730-done-qualifier` | o pronto como régua do turno do filho |
| gravação de sessões | `Record` e `Rebuild` de `internal/session/` |
| `gh` no PATH, com login | só para o forge `github`; sem ele, `local`, dito |

## Fora desta fatia, com ponteiro

| O quê | Regra do `.r` | Onde volta |
|---|---|---|
| daemon como LaunchAgent do `launchd`, emenda de D19 | RN-2, Q-C | passo 11 do `.r` §10, na segunda fatia |
| supervisionado, autônomo, a troca de nível, `autonomy.changed` | RN-3 | segunda fatia; `.config` §5 |
| a lista "sempre perguntar" como negação de `internal/policy`, `always_ask` | RN-4 | segunda fatia |
| tetos de custo e de tempo; parar filho vivo com `budget` ou `deadline` | RN-3 | segunda fatia; `.config` §5 — o de tokens é desta, passo 6 |
| custo em dinheiro na lista, e a tabela de preços | RN-9 | segunda fatia, com o teto de custo (decisão 1) |
| deploy declarado, `deploy.*` | RN-13 | segunda fatia |
| notificação do macOS e do celular | RN-10, Q-B | segunda fatia |
| os sete contratos com modelo real (`make eval`) | §9 do `.r` | segunda fatia, depois do log lido |
| arquivar a orquestradora, que encerra os filhos | RN-7 | segunda fatia |

## Ordem de execução

### Passo 2 — `feat(protocol): a session can have a parent`

Em `internal/protocol/` (arquivo novo `orchestrate.go`) e `pkg/client/`; tipos da `.p` §5.2,
eventos da §5.3, códigos da §5.4. Nenhuma rota responde ainda.

- [ ] `Session` e `Conversation` com `ParentID`, `ProjectID`, `Role`, `Owns`, todos `omitempty`.
- [ ] `TurnStarted.Origin`, `OriginChildren`, `StopDaemonRestart`.
- [ ] `Project`, `Child`, `Return`, `Verdict`, `Finding`, `GateResult`, `ChildPR`, `MainHealth`,
      `Question`, `QuestionOption` e os payloads dos eventos.
- [ ] Os doze códigos com o status da §5.4 em `httpStatus`.
- [ ] `desktop/src/protocol/generated.ts` regenerado pelo `tygo` no mesmo PR.
- [ ] `client-server-protocol.p` §5 e §8 e changelog dessa família; changelog da
      `delegated-writing` dizendo que a RN-3 dela ("filho não pergunta") não vale para o filho-sessão.

**Testes obrigatórios:**

- Golden de uma `Session` sem os campos novos: os bytes são os de hoje (aditivo de verdade).
- Todo código novo tem status em `httpStatus` (o guarda de hoje cobre, e precisa ver os doze).

### Passo 3 — `feat(session): a recorded session reopens under its own id`

Em `internal/session/`. A orquestradora e os filhos precisam voltar com o id que os outros guardam.

- [ ] `Manager` ganha reabrir: id do registro, histórico por `Rebuild`, `Seq` continuando do último
      evento gravado, o arquivo aberto em acréscimo (`NewRecord` já abre assim).
- [ ] Turno que o registro mostra aberto ganha `turn.completed{reason: daemon_restart}` antes de
      qualquer evento novo.
- [ ] `SubmitOrigin(text, origin)`: o turno nasce com `TurnStarted.Origin`.
- [ ] Aprovador sem prazo: `ExpiresAt` zero, sem o `ApprovalTimeout`.

**Testes obrigatórios:**

- Reabrir uma sessão gravada devolve o mesmo id, o mesmo histórico e o próximo `Seq` sem buraco.
- Sessão gravada no meio de um turno reabre com `daemon_restart` como último fim de turno.
- Aprovação sem prazo continua pendente depois de `ApprovalTimeout`.

### Passo 4 — `feat(orchestrate): every decision is logged with its reason`

Pacote novo `internal/orchestrate/`: `DecisionLog`, `Decision`, o esquema da `.p` §9.1. Lê
`orchestrator.dir`.

- [x] `Record` escreve uma linha, com `fsync`, e devolve o `seq`; `why` vazio é erro.
- [x] `seq` continua do último do arquivo ao reabrir; linha final cortada por uma queda é cortada
      e devolvida por `Torn`, e linha completa ilegível ou `seq` fora de ordem impede o log de abrir.
- [x] Linha acima de 64 KiB corta o campo maior e diz quanto.

**Testes obrigatórios:**

- Decisão sem `why` é recusada e nada é escrito.
- Reabrir o log continua o `seq` sem buraco.
- Escrita que falha (diretório só leitura) é erro devolvido, nunca uma linha perdida em silêncio.

### Passo 5 — `feat(orchestrate): a project has one orchestrator that survives the daemon`

`Registry`, `Orchestrator`, `Store`, `Spawner`; rotas `POST /v1/projects`, `GET /v1/projects`,
`GET /v1/projects/{id}`, `GET /v1/projects/{id}/orchestrator` em `internal/server/`; `app.Daemon`
implementa a `Spawner`. Lê `orchestrator.enabled`, `orchestrator.profile`,
`orchestrator.main_branch`.

- [x] `ProjectID` da raiz resolvida da worktree principal (`--git-common-dir`); outra worktree da
      mesma árvore dá o mesmo projeto.
- [x] A orquestradora é sessão `RoleOrchestrator`, modo `plan`, na raiz.
- [x] `state.json` atômico; `Reload` no boot reabre a orquestradora sob o mesmo id.
- [x] `record.enabled` falso ou `orchestrator.enabled` falso: `orchestrator_unavailable` dizendo qual.

**Testes obrigatórios:**

- Dois `POST /v1/projects`, um pela raiz e outro por um link ou por uma worktree, dão o mesmo `Project`.
- Daemon que reinicia devolve a mesma orquestradora, com o histórico.
- `state.json` ilegível: o projeto não abre e o erro é devolvido por `Reload`.

### Passo 6 — `feat(orchestrate): a child works in its own worktree`

`Leases`, `OwnsOverlap`, o trecho de `worktree.go` que cria e remove a worktree do filho, as
ferramentas `delegate`, `children`, `steer_child`, `close_child` em `internal/tools/` (ao lado de
`internal/tools/explore.go`), `GET /v1/sessions/{id}/children`, `DELETE` de filho, `child.started`,
`child.state`, `child.completed`, o teto de tokens (`.p` §10). Estados `dispatched`, `working`,
`awaiting_person`, `failed`, `closed`. Lê `max_children`, `child_profiles`, `branch_prefix`,
`closed_retention`, `token_ceiling`.

- [x] `delegate` tudo ou nada, validando o lote antes da primeira worktree.
- [x] Worktree a partir da ponta da main, em `children/<id>/`, branch `<prefix><slug>-<id6>`, os seis **últimos** do id (os primeiros são o relógio e se repetem num lote).
- [x] `core.hooksPath` lido antes e depois; diferença vira `hooks.changed`.
- [x] Registro do filho sem as ferramentas da `.p` §10; a regra de negação de git que grava.
- [x] Commit do harness ao fim de cada turno, com os caminhos nomeados (a conferência de `owns`
      entra no passo 8; aqui, o commit).
- [x] Fechado sai da resposta de `/children` depois de `closed_retention`.
- [x] `State.Spent` soma os tokens de toda sessão do projeto; no teto, `delegate` é recusado com
      `ceiling_reached`, `ceiling.reached` vai ao log, a conversa diz, e `Question{kind: ceiling}`
      oferece `continue` (zera a janela) ou `stop`. Filho vivo não é parado. A resposta é
      `Orchestrator.Answer`; a rota de `/questions` é do passo 10.

**Testes obrigatórios:**

- Lote com dois itens cujo `owns` se cruza: `owns_overlap` com os dois caminhos, e nenhuma worktree.
- Lote contra filho vivo com `owns` cruzado: recusado inteiro.
- Lote acima de `max_children`: `max_children_reached`, nada criado.
- Perfil fora de `child_profiles`: `model_not_allowed`.
- O registro do filho não tem `explore`, `delegate`, `children`, `steer_child`, `close_child`, `ask`.
- `git push` e `git rebase` no `bash` do filho são negados sem pedido de aprovação, com o porquê.
- Pedido de aprovação do filho fica pendente além de `ApprovalTimeout`, e o filho fica
  `awaiting_person`.
- Criar filhos volta com os ids e a orquestradora fica `idle` com eles rodando.
- `Spent` no teto: `delegate` recusado com `ceiling_reached`, nenhuma worktree, `ceiling.reached` no
  log; `continue` zera e o próximo lote passa.
- `OwnsOverlap` em tabela: prefixo contém prefixo, separador, metacaractere, o caso conservador.

### Passo 7 — `feat(orchestrate): a child's report starts the orchestrator's turn`

A caixa de relatos (`State.PendingReports`), o turno `Origin: children`, o debounce. Lê
`orchestrator.report_debounce`.

- [x] Relato com a orquestradora `idle` abre um turno depois do debounce, juntando os que chegaram.
- [x] Relato com turno em curso espera o fim dele; nunca entra no meio.
- [x] Texto do relato montado pelo harness, em inglês, acrescentado ao fim do histórico.

**Testes obrigatórios:**

- Dois filhos que terminam dentro do debounce viram **um** turno com os dois relatos.
- Relato durante um turno da pessoa entra só no começo do turno seguinte.
- O prefixo do contexto antes do relato é byte a byte o de antes (ADR-03).

### Passo 8 — `feat(orchestrate): a reviewer reads every child's diff`

A conferência de `owns` (`.p` §7.1), a sessão `RoleReviewer` e a ferramenta `verdict` (§7.2), a
devolução em duas fases com `red_test` (§7.3), `review.verdict`. Estado `in_review`. Lê
`reviewer_profiles`, `review_max_diff_bytes`, `max_returns`.

- [x] Caminho fora do `owns` no `git status --porcelain` devolve sem commit e sem revisão.
- [x] Revisor só com `read`, `glob`, `grep`, `symbol`, `verdict` (`ReviewerRegistry`).
- [x] `missing` roda de novo uma vez; o segundo fica em `in_review` com relato.
- [x] `changes_requested` → fase `red` → `red_test.confirmed` → fase `fix` → 7.1 de novo.
- [x] Além de `max_returns`, `failed` com `incomplete`.

**Testes obrigatórios:**

- Diff com um arquivo fora do `owns`: `Return{outside_owns}`, sem commit, sem sessão de revisor.
- O registro do revisor não tem ferramenta que escreve. **Feito**:
  `TestTheReviewerRegistryHasNoToolThatWrites`; e `TestADiffOutsideOwnsReturnsWithoutAReview`, sem
  revisor, em `internal/orchestrate/review_test.go`.
- Teste da fase `red` que passa não é aceito como vermelho, e o filho volta à fase `red`.
- Conserto entregue sem a fase `red` confirmada não segue para o gate.

### Passo 9 — `feat(orchestrate): the orchestrator runs the gate itself`

O lugar de integração, a atualização com merge da main, o conflito devolvido, `Gate.Measure`,
`gate.measured`. Lê `gate_command`, `gate_timeout`, e `verify.command` como queda.

- [x] Integração em `integration/`, `--detach`, criada uma vez por projeto (`OpenIntegration`, passo 10).
- [x] `git merge --no-ff M` sobre `Head`; conflito abortado na integração e refeito na worktree do
      filho com os marcadores; `Return{conflict, paths}` (`Integration.Update`, passo 10); o commit do
      harness ao fim do turno seguinte conclui o merge.
- [x] Gate em `H` numa worktree descartável, sem `GIT_*`, com tempo-limite; saída inteira em `gates/<commit>.log`
      (`Gate.Measure`); `gate.measured` com a alegação do filho como `claim` (`RecordGate`); `gate_stale`
      conferido por `CheckFresh`. A esteira o chama depois da aprovação; o sandbox do `bash` em volta do
      gate continua por fazer.
- [ ] `vcs` sem nenhuma função de rebase ou de push forçado.

**Testes obrigatórios:**

- `gate.measured.commit` é o `H` que contém a ponta da main, e `base` é essa ponta.
- Gate vermelho devolve ao mesmo filho em fase `fix`, com a saída.
- Sem `gate_command` nem `verify.command`: `gate_undeclared`, nada segue. **Feito**:
  `TestAnUndeclaredGateIsSaidAndNothingRuns`, com verde, vermelho, `timeout`, alegação contra medição,
  `gate_stale` e limpeza em `internal/orchestrate/gate_test.go`.
- Conflito volta ao filho dono da branch com os caminhos; a integração termina limpa.
- Guarda de fonte: nenhum `rebase`, `--force`, `--amend` em `internal/vcs/` nem em `internal/orchestrate/`.

### Passo 10 — `feat(orchestrate): a merge waits for the person`

A fila de perguntas (`ask`, `GET` e `POST` de `/questions`), o PR pelo `Forge`, a pergunta de merge,
a fila serial, `gate_stale`, `main.health`, o forge `local`. Estados `pr` e `merged`. Lê `forge`,
`ci_timeout`, `sensitive`, `sensitive_diff_lines`.

- [x] Gate verde: push sem `--force`, PR com bloco de evidência, `Question{kind: merge}` (`Merges.Propose`).
- [x] Só a pessoa responde; o primeiro que responde leva (`Merges.Answer`, `POST /questions`); a
      ferramenta `ask` pergunta, e a fila fica no `State`.
- [x] Merge: main vermelha recusa; `gate_stale` mede de novo; `gh pr merge --squash
      --match-head-commit`; local por `update-ref` condicional (`Merges.Merge`).
- [x] Depois do merge, `main.health` até verde, vermelho ou `ci_timeout`.

**Testes obrigatórios** (com o `gh` falso de `internal/vcs/pr_test.go`):

- O `gh pr merge` chamado leva `--match-head-commit` com o `GateResult.Commit`.
- Ponta da main que andou depois do gate: `gate_stale`, nenhum merge, filho de volta a `in_review`.
- Main vermelha: o segundo merge da fila é `merge.refused{main_red}`; `fixes_main` passa.
- Local com a main em checkout numa worktree: recusado, dizendo onde.
- Local: `update-ref` com o valor antigo errado não move a main.
- `question_already_answered` para a segunda resposta.
- **Feitos**, com um `gh` falso próprio (`internal/orchestrate/testdata/fakegh`), em
  `merge_test.go`, `merge_local_test.go`, `integrate_test.go` e `forge_test.go` de
  `internal/orchestrate/`; e mais: o merge que espera o sim, a recusa da pessoa com o porquê, o
  conflito com os caminhos e o `pending` que vira `unknown`.

### Passo 11 — `feat(cli): dcode decisions reads the log`

`cmd/dcode/` ganha `decisions` (arquivo novo `decisions.go`), da `.p` §9.2.

**Testes obrigatórios:**

- Lê o log com o daemon desligado.
- Linha ilegível é dita em stderr com o número, o resto é impresso, saída 1.
- `seq` com buraco é dito.
- `--kind` e `--child` filtram; `--json` devolve as linhas byte a byte.

### Passo 12 — `test(orchestrate): scripted scenarios`

Os cenários da RN-11, em Go, no pacote `internal/orchestrate/` (arquivo `scenario_test.go`): um
repositório de brinquedo por cenário, um `origin` bare local, o `gh` falso de
`internal/vcs/pr_test.go`, e o modelo roteirizado de `internal/app/e2e_test.go` (`scriptedModel`)
respondendo pela orquestradora, pelos filhos e pelo revisor. De graça, determinísticos, em todo PR
— a régua do `check:daemon` do desktop (`desktop/tests/daemon/scenarios.mjs`) aplicada ao núcleo.

| Teste | O que prova |
|---|---|
| `TestScenarioThreeThemesBecomeThreeChildren` | lote de três, `owns` disjuntos, três PRs, merge um por vez depois do sim |
| `TestScenarioAChildThatClaimsTheGatePassedIsMeasured` | o filho diz "make check passed"; o gate real é vermelho; nenhum PR; `claim` e `measured` lado a lado no log |
| `TestScenarioAMissingHookDoesNotSkipTheGate` | `core.hooksPath` aponta para o nada; o gate roda igual; `hooks.changed` quando muda |
| `TestScenarioARedGateReturnsToTheSameChild` | gate vermelho volta ao mesmo filho, o conserto passa, o merge segue |
| `TestScenarioAReviewDefectReturnsRedFirst` | o revisor aponta; fase `red` confirmada antes do conserto |
| `TestScenarioAConflictReturnsToTheChildThatOwnsTheBranch` | a main andou e conflita; volta ao filho; ele resolve; mede de novo |
| `TestScenarioADaemonRestartEndsTheTurnWithItsReason` | daemon cai no meio de um turno de filho; volta com `daemon_restart`, mesma orquestradora, mesmo `seq` |
| `TestScenarioMainMovedBetweenGateAndMerge` | outro merge entra entre o gate e o sim; `gate_stale`; mede de novo |
| `TestScenarioARedMainStopsTheQueue` | CI da main vermelho depois do merge; o próximo é recusado |
| `TestScenarioADiffOutsideOwnsReturnsWithoutReview` | o filho escreve fora do `owns` pelo `bash`; volta sem revisão |
| `TestScenarioNoForgeMergesLocally` | sem `origin`: merge local, `forge.local` dito uma vez |
| `TestScenarioAChildApprovalWaitsForThePerson` | aprovação de filho não expira e a orquestradora segue com os outros |

Cada cenário termina lendo o `decisions.jsonl` e conferindo os `kind` esperados, em ordem.

### Passo 13 — `feat(tui): the orchestrator lists its agents`

Em `internal/tui/` (arquivo novo `agents.go`) e `cmd/dcode/`. `dcode orchestrate` abre a
orquestradora do projeto corrente pelo `POST /v1/projects`; `dcode` sozinho continua a conversa
comum (decisão 4). A coluna lateral (`side.go`) troca o
diff pela lista: `● índice  trabalhando  41k tok  #518 ✓`, e `◆ 2 perguntas esperam você`. `^Q` abre
a pergunta mais antiga, com as opções e a recomendada marcada; Enter responde. `^R` lista os filhos
sob a orquestradora; escolher um anexa a TUI a ele, onde a aprovação é respondida.

**Testes obrigatórios:**

- Golden da coluna com um filho em cada um dos oito estados.
- `child.state` muda a linha sem reler `/children`.
- Fechado some quando a resposta do daemon não o traz mais; `failed` fica.
- `^Q` responde a pergunta pelo `POST`, e `question_already_answered` é dito.

**Feito na primeira parte** (a lista, sem tecla nova): os três primeiros testes acima —
`TestTheColumnDrawsAChildInEachOfTheEightStates`,
`TestAChildStateChangesItsRowWithoutRereadingTheList` e
`TestAClosedChildLeavesWhenTheDaemonNoLongerListsItAndAFailedOneStays`, em
`internal/tui/agents_test.go`. A forma da coluna está na `client-tui.p` §2.2. O do `^Q` vem com a
tecla.

**Feito na segunda parte**, com o resto do passo:

- [x] `dcode orchestrate` resolve o projeto da pasta pelo `POST /v1/projects` e anexa à
      orquestradora (`GET .../orchestrator`), num daemon que já roda: nunca sobe um próprio, que
      morreria com o terminal. Sem daemon, diz `dcode serve`; desligada, diz `orchestrator.enabled`
      e `record.enabled`. `dcode` sozinho não muda (decisão 4).
- [x] Ao abrir, `/children` e `/questions` por `loadAgents`, depois do histórico.
- [x] `^Q` abre a pergunta mais antiga com a recomendada marcada; Enter responde pelo `POST`;
      `question_already_answered` é dito — `TestASecondAnswerIsSaidAsAlreadyAnswered`, o quarto
      teste obrigatório, com os outros em `internal/tui/orchestrate_test.go`.
- [x] `^R` lista os filhos e anexa ao escolhido; do filho, a orquestradora é a primeira linha.
- [x] Testes do comando com um daemon falso em `cmd/dcode/orchestrate_test.go`.

O porquê está no changelog da `client-tui`, `202610181200-dcode-orchestrate-abre-a-orquestradora.md`.

### Passo 14 — `feat(desktop): the live agent list`

`AgentsPanel` no painel lateral da conversa orquestradora, com a gramática da **Mudanças** (D41):
uma linha por filho, a marca do estado, título, estado, tokens; PR e CI à direita como D60 e D61.
No topo, "passo a passo" e "◆ N perguntas", que abre a fila. Clicar abre o filho na grade; **a
grade não muda**, e as telas de referência 02 e 03 continuam a 0,00%. Só o fio (D8). D-número no
PR; fragmento em `desktop/changelog.d/`. O desktop ganha a opção de abrir a orquestradora do
projeto, ao lado da conversa comum, que continua o padrão (decisão 4).

**Testes obrigatórios:**

- Redutor de estado: `child.*` e `question.*` sobre a lista inicial dão a lista esperada.
- Cenário novo em `desktop/tests/daemon/` (`check:daemon`): dois filhos aparecem, um fica
  "esperando você" com o pedido do filho, e o merged sai depois da retenção curta do teste.
- `check:visual` das telas 02 e 03 sem diferença.

### Fechamento — as invariantes deixam de ser previstas

**Feito no passo 4**, que testou as linhas do log: a `.p` §13 é "Invariantes verificáveis", a §14
as previstas, e `internal/orchestrate/invariants_test.go` nomeia `orchestration`. Cada passo
seguinte sobe para a §13 as linhas a que dá teste.

A regra, como foi escrita: no primeiro PR acima que tiver teste para uma linha da `.p` §13, a seção
passa a se chamar "Invariantes verificáveis" só com as linhas que já têm teste — as outras ficam
numa lista "previstas" abaixo dela — e um `invariants_test.go` no pacote passa a nomear
`orchestration` no `specguard.Check`, como `internal/loop/invariants_test.go` faz para as outras
famílias. No passo 12, nenhuma linha fica prevista.

## Diferenças da ordem do `.r` §10

- **3 e 7 são PRs próprios.** Reabrir sob o mesmo id muda o `session.Manager`, que todo o resto
  usa; o relato muda como um turno começa. Juntos com outro passo, o título pediria "and".
- **8, o revisor, ganhou passo.** O `.r` o deixava dentro do gate; são medições diferentes, com
  sessões diferentes, e o gate não deve depender de quem lê.
- **11, o comando, saiu do log.** É cliente do log, e o log é útil antes dele.
- **`launchd`** (passo 11 do `.r`) é da segunda fatia.

## Decisões do dono, 2026-10-17

As seis perguntas deste plano, respondidas pelo dono com a recomendação de cada uma; o porquê está
no changelog da família (`202610172200`).

1. **Custo na lista é tokens** — entrada, saída e cache — nesta fatia. Dólares e a tabela de preços
   entram na segunda, com o teto de custo, que precisa da mesma tabela.
2. **Revisão, gate e merge são passos do harness**, que correm sempre e em ordem quando um filho
   termina (`.p` §7); o modelo lê os resultados e decide. Isto muda a redação do `.r` §6, que os
   chamava de ferramentas: um modelo que chama o gate pode não chamar.
3. **A orquestradora roda em modo `plan` e nunca escreve na árvore**; só filhos escrevem, nas
   worktrees deles.
4. **A entrada é `dcode orchestrate`** nesta fatia, e o desktop ganha a opção; `dcode` continua a
   conversa comum. O padrão troca depois de o log da primeira fatia ser lido.
5. **O teto de tokens é desta fatia**, no passo 6, que despacha filhos: chegar nele para o despacho
   e diz por quê. `orchestrator.token_ceiling` é lida no passo 6. Tetos de custo e de tempo ficam na
   segunda.
6. **O harness faz o git de cada filho** — commit (teste vermelho, conserto), merge da main e push
   —, e os comandos git que gravam são negados ao filho com o porquê.
