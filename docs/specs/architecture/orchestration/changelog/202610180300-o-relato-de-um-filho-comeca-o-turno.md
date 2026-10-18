# O relato de um filho começa o turno da orquestradora

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §7.1, §7.3, §13, §14; `.config` §2; `.i`,
passos 7, 8 e 9).

## O que mudou

O passo 7 da primeira fatia, e a fiação dos componentes que os passos 8 e 9 deixaram sem chamador,
pela decisão 2 do dono: revisão, gate e merge são passos do harness, que correm sempre e em ordem; o
modelo lê os resultados e decide.

- **A esteira** (`internal/orchestrate/pipeline.go`, `pipeline_gate.go`). Ao fim de cada turno de
  filho, o `git status` é conferido contra o `owns` **antes** do commit (`owns.checked`); fora dele,
  nada é commitado e o filho volta com `Return{outside_owns, paths}`, sem revisor. Turno `done` vai a
  `in_review`: `AssembleReview` sobre `Base..Head`, uma sessão revisora (`Spawner.SpawnReviewer`) com o
  `ReviewerRegistry`, o `Prompt` como primeiro turno, e o `Finish` ao fim do turno dela, com
  `review.verdict` no log e no fio. `missing` roda de novo uma vez; o segundo deixa o filho em
  `in_review`, relatado. `approve` mede o gate; `changes_requested` devolve em fase `red`.
- **A fase `red`.** O filho escreve só o teste e o nomeia com `red_test{command}`
  (`internal/tools/redtest.go`); ao fim do turno o harness commita e roda o comando nesse commit, numa
  worktree descartável: saída diferente de zero é `red_test.confirmed` e a fase `fix`; verde, ou turno
  sem `red_test`, volta a `red`, dizendo. A `fix` termina e a esteira recomeça na conferência de `owns`.
- **O gate.** Depois da aprovação, `Gate.Measure` no `Head` exato, com `Base` a ponta da main quando o
  `Head` a contém; `RecordGate` com a conclusão do filho como `claim`; `gate.measured` no fio. Vermelho
  ou `timeout` devolve em fase `fix`, com o fim da saída; `undeclared` é relatado e nada segue; verde
  vai à `MergeQueue`, e o filho vai a `pr` com o PR que ela abriu.
- **A caixa** (`internal/orchestrate/reports.go`). Todo desfecho — devolução, falha, revisão sem
  veredito, gate, `daemon_restart`, a resposta da pessoa a uma pergunta — vira um relato em
  `State.PendingReports`, em disco. Depois de `orchestrator.report_debounce` a caixa vira **um** turno
  da orquestradora com `Origin: children`; com um turno em curso, `turn_already_active` deixa a caixa
  como está e o fim do turno a agenda de novo. No boot, a caixa pendente é entregue depois da reabertura.
- Passam a ser lidas `orchestrator.max_returns` e `orchestrator.report_debounce` (`ReadLimits`), e o
  gate e a revisão (`GateFrom`, `ReviewFrom`) quando o projeto abre.

## O que o código decide onde a `.p` deixava a forma em aberto

- **`State.Briefs`.** O `Child` do fio não carrega a tarefa; o revisor precisa dela, e o passo de merge
  do `pr_title` e do `pr_body`. O `State` ganha `Briefs` por filho — tarefa, texto do PR,
  `fixes_main`, a última conclusão do filho (o `claim` do gate) e o `red_test` pendente.
- **O id do revisor nasce antes da sessão**, como o do filho: `ReviewRequest` ganha `ID` e `Submit`,
  porque o `Review` que o `verdict` alimenta é nomeado pela sessão que o dá.
- **`red_test` está no registro de todo filho e recusa fora da fase `red`**, dizendo por quê. A `.p`
  dizia "só na fase `red`"; o registro de uma sessão é montado uma vez, e trocá-lo no meio da conversa
  mudaria o prefixo que o modelo já leu (ADR-03).
- **A passagem de `red` a `fix` não conta em `max_returns`**: é a segunda metade da mesma devolução.
  `red` sem `red_test`, ou com teste que passa, conta.
- **Turno `done` sem commit nenhum** vai a `failed`, `incomplete`, relatado: não há o que revisar.
- **`merge.queued` não é da esteira.** Pela §7.6 é a resposta `merge` da pessoa; o passo 10 a escreve.
  A esteira entrega o filho verde pela `MergeQueue` (`Ready`) e o move a `pr` com o PR que volta; sem
  ela, o filho fica em `in_review` com a medição e o relato diz isso.
- **`Base` quando a main andou.** Sem a integração (`integration/`, o merge da main), o `Head` pode não
  conter a ponta da main; aí `Base` é o `merge-base`, honesto, e o relato diz que o merge mede de novo.
  A `CheckFresh` do passo 10 recusa essa medição com `gate_stale`.
- **Os passos longos** — gate, `red_test` — correm fora da goroutine que lê os eventos da sessão, para
  que um gate de meia hora não derrube a assinatura do log.

## Invariantes e o que fica

Sobem para a §13 três linhas da §14 — o diff fora do `owns` sem commit e sem revisão, o defeito em fase
`red` antes do conserto, o relato só no começo de um turno com `Origin: children` — e entram cinco
novas, cada uma com o teste que a reivindica. Ficam na §14: o gate no `H` que **inclui** a main de
agora, e o resto do merge.

Fica para a fiação do passo 10: a integração (`OpenIntegration`, `Update`) antes do gate, o conflito
devolvido ao filho com os marcadores, a `MergeQueue` real (`Propose`: push, PR, `Question{merge}`), a
rota `/questions`, e o sandbox do `bash` em volta do gate. A esteira não reabre revisores no boot: a
revisão em curso roda de novo, como a §8 pede.
