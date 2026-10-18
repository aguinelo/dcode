# O relato de um filho começa o turno da orquestradora

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §7.1, §7.3, §13, §14; `.config` §2, §3; `.i`,
passos 7 a 10).

## O que mudou

O passo 7 da primeira fatia, e a fiação dos componentes que os passos 8, 9 e 10 deixaram sem
chamador, pela decisão 2 do dono: revisão, gate e merge são passos do harness, que correm sempre e em
ordem; o modelo lê os resultados e decide.

- **A esteira** (`internal/orchestrate/pipeline.go`). Ao fim de cada turno de filho, o `git status` é
  conferido contra o `owns` **antes** do commit (`owns.checked`); fora dele, nada é commitado e o filho
  volta com `Return{outside_owns, paths}`, sem revisor. Turno `done` vai a `in_review`:
  `AssembleReview` sobre o trabalho do filho, uma sessão revisora (`Spawner.SpawnReviewer`) com o
  `ReviewerRegistry`, o `Prompt` como primeiro turno, e o `Finish` ao fim do turno dela, com
  `review.verdict` no log e no fio. `missing` roda de novo uma vez; o segundo deixa o filho em
  `in_review`, relatado. `approve` segue para o gate; `changes_requested` devolve em fase `red`.
- **A fase `red`** (`pipeline_gate.go`). O filho escreve só o teste e o nomeia com
  `red_test{command}` (`internal/tools/redtest.go`); ao fim do turno o harness commita e roda o comando
  nesse commit, numa worktree descartável: saída diferente de zero é `red_test.confirmed` e a fase
  `fix`; verde, ou turno sem `red_test`, volta a `red`, dizendo. A `fix` termina e a esteira recomeça
  na conferência de `owns`.
- **A atualização e o gate.** Depois da aprovação, `Integration.Update` faz o merge da ponta da main na
  branch, no lugar de integração, e a branch do filho anda até `H` (`branch.updated`); conflito volta
  ao filho com os marcadores na worktree dele, e o commit do harness ao fim do turno seguinte conclui
  o merge. `Gate.Measure` em `H`, `Base` a ponta da main; `RecordGate` com a conclusão do filho como
  `claim`; `gate.measured` no fio. Vermelho ou `timeout` devolve em fase `fix`, com o fim da saída;
  `undeclared` é relatado e nada segue.
- **O merge** (`merging.go`). Verde: `Merges.Propose` empurra a branch, abre o PR e faz a
  `Question{merge}`; o filho vai a `pr`, esperando `merge`. Só a resposta da pessoa —
  `POST /v1/sessions/{id}/questions/{qid}`, o primeiro que responde leva — põe o filho na fila
  (`merge.queued`, escrito pelo `Merges.Answer`); a fila corre um por vez (`Merges.Merge`), e o filho
  vai a `merged`, com a sessão fechada e a worktree removida. `return` devolve ao filho
  (`Return{person}`); `close` o fecha; `gate_stale` o leva de volta a `in_review`, medido de novo;
  main vermelha para a fila.
- **As perguntas** (`questions.go`, `internal/tools/ask.go`). A orquestradora ganha `ask`, que põe uma
  pergunta de desenho na fila e volta na hora; `GET /v1/sessions/{id}/questions` lista as perguntas do
  projeto, e o `POST` as responde — só a pessoa.
- **A caixa** (`reports.go`). Todo desfecho — devolução, falha, revisão sem veredito, gate, pergunta
  respondida, merge, `daemon_restart` — vira um relato em `State.PendingReports`, em disco.
  Depois de `orchestrator.report_debounce` a caixa vira **um** turno da orquestradora com
  `Origin: children`; com um turno em curso, `turn_already_active` deixa a caixa como está e o fim do
  turno a agenda de novo. No boot, a caixa pendente é entregue depois da reabertura.
- **O que sobrevive ao reinício**: as perguntas de merge, a fila, a última leitura da main e
  `LocalSaid` ficam no `State`; o `forge.local` é dito uma vez por projeto, e a fila retoma no boot.
- Passam a ser lidas `orchestrator.max_returns` e `orchestrator.report_debounce` (`ReadLimits`), e o
  gate, a revisão e o merge (`GateFrom`, `ReviewFrom`, `MergeFrom`) quando o projeto abre.

## O que o código decide onde a `.p` deixava a forma em aberto

- **`State.Briefs`.** O `Child` do fio não carrega a tarefa; o revisor precisa dela, e o merge do
  `pr_title`, do `pr_body` e do `fixes_main`. O `State` ganha `Briefs` por filho — esses, a última
  conclusão do filho (o `claim` do gate) e o `red_test` pendente.
- **O id do revisor nasce antes da sessão**, como o do filho: `ReviewRequest` ganha `ID` e `Submit`,
  porque o `Review` que o `verdict` alimenta é nomeado pela sessão que o dá.
- **`red_test` está no registro de todo filho e recusa fora da fase `red`**, dizendo por quê. A `.p`
  dizia "só na fase `red`"; o registro de uma sessão é montado uma vez, e trocá-lo no meio da conversa
  mudaria o prefixo que o modelo já leu (ADR-03).
- **A passagem de `red` a `fix` não conta em `max_returns`**: é a segunda metade da mesma devolução.
  `red` sem `red_test`, ou com teste que passa, conta.
- **Turno `done` sem commit nenhum** vai a `failed`, `incomplete`, relatado: não há o que revisar.
- **A revisão lê desde o `merge-base` com a main**, não desde o `Base` da branch: depois de a
  integração trazer a main, o que veio dela não é do filho, nem para o revisor nem para o `owns`.
  No turno que resolve um conflito, caminho igual ao `MERGE_HEAD` também não conta contra o `owns`.
- **O passo de merge abre na primeira vez que um filho precisa dele**, não no `POST /v1/projects`: o
  lugar de integração é uma worktree, e o forge pergunta ao `gh`. Por isso `github` declarado sem `gh`
  é dito ali, como `stall` relatado, e não na abertura do projeto (emenda do `.config` §3).
- **Uma transição tardia não move um filho terminal.** Um passo da esteira que termina depois de a
  pessoa fechar o filho não o reabre: `move`, a devolução e a proposta conferem o estado antes.
- **O lugar de integração com link no caminho** (`/var` é `/private/var` no macOS) era lido como
  "não é a worktree de integração" ao reabrir; `OpenIntegration` compara os caminhos resolvidos.
- **A ponta da main é a do forge** (`Forge.MainTip`). Com `github`, `git fetch origin <main>` e
  `refs/remotes/origin/<main>`; com `local`, `refs/heads/<main>`. A esteira mede, revisa e atualiza
  sobre ela, que é a mesma que o merge compara: medir sobre a main local, que um merge no GitHub nunca
  move, deixava todo filho seguinte `gate_stale` para sempre, achado pelos cenários do passo 12.
  `fetch` que falha é dito e para o passo, nunca a main local no lugar.
- **Filho com PR mantém o PR** (`Forge.UpdatePR`, `pr.updated`, kind novo da §9.1). Medido de novo
  depois de `gate_stale`, a branch é empurrada de novo, sem `--force`, e o corpo do PR é refeito com a
  evidência nova; pedir um segundo PR ao `gh` falhava e parava a esteira.
- **`child.returned` tem uma forma só**, `{return, returned}`, escrita sempre pela devolução da
  esteira; a integração devolve o `*Conflict` e não escreve mais a linha dela (§9.1).
- **Os passos longos** — gate, `red_test`, a fila — correm fora da goroutine que lê os eventos da
  sessão, para que um gate de meia hora não derrube a assinatura do log.

## Invariantes e o que fica

Sobem para a §13 quatro linhas da §14 — o diff fora do `owns` sem commit e sem revisão, o defeito em
fase `red` antes do conserto, o gate no `H` que inclui a main de agora, o relato só no começo de um
turno com `Origin: children` — e entram cinco novas, cada uma com o teste que a reivindica. Ficam na
§14: todo fim de turno com um nome da RN-7, a guarda de fonte contra `rebase`, e toda transição com
linha antes do evento.

Fica por fazer: o sandbox do `bash` em volta do gate; a conciliação pelo forge de um merge em curso
no reinício (`gh pr view`); os cenários roteirizados do passo 12.
