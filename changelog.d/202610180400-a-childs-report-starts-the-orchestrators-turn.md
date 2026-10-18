## en

- **A child's report starts the orchestrator's turn, and the pipeline runs on its own.** When a
  child's turn ends, the harness first checks `git status` against the child's owns. A change outside
  them is committed by nobody, reviewed by nobody, and handed back (`outside_owns`). Inside, the harness
  commits by name. If the turn ended done, a reviewer session follows. It runs in plan mode in the
  child's worktree, with only `read`, `glob`, `grep`, `symbol` and `verdict`. A review that ends
  without a verdict runs once more, then stays `in_review` and is reported. `changes_requested` goes
  back to the same child in two phases. In the red phase the child writes only the failing test and
  names it with the new `red_test` tool. The harness runs it on that commit before any fix, and takes
  the fix only once the test fails (`red_test.confirmed`). After an approval the harness merges main's
  tip into the branch in its integration place. With forge github that tip is origin's, fetched first. A conflict goes back to the child with the markers,
  and the harness's next commit concludes the merge. The harness then measures the gate itself on the
  result, with main's tip as its base, and keeps the child's own claim beside the measurement. A red
  gate goes back in the fix phase. A green one is pushed, its pull request opened, and the person
  asked. A child measured again after `gate_stale` keeps its pull request: the branch is pushed again,
  never forced, and the evidence refreshed. Only the person's `merge`, through the new `POST /v1/sessions/{id}/questions/{qid}`, puts the
  child in the serial queue. The child then merges, its worktree is removed, and it is `merged`.
  `return` hands the work back and `close` closes it. The orchestrator gains `ask` for design
  questions, and `GET /v1/sessions/{id}/questions` lists them. Past `orchestrator.max_returns` (3) the
  child fails `incomplete`. `review.verdict`, `gate.measured`, `question.*`, `merge.*` and
  `main.health` reach the orchestrator's wire. Every outcome, and every answer, goes into the
  orchestrator's inbox. The inbox is kept on disk with the merge questions and the queue, all
  delivered after a restart. Reports that arrive within `orchestrator.report_debounce` (2s) start one
  orchestrator turn with origin `children`. That turn never starts in the middle of another, and it is
  appended so the model's context before it is unchanged. Steps 7 to 10 of the orchestrator's first
  slice, wired. Before, those steps were components with no caller: a child that finished stayed
  `working`, and nothing reviewed, measured or merged it.

## pt-BR

- **O relato de um filho começa o turno da orquestradora, e a esteira corre sozinha.** Ao fim de um
  turno de filho, o harness confere o `git status` contra o `owns` antes de tudo: mudança fora dele não
  é commitada nem revisada, e volta ao filho (`outside_owns`); dentro, o harness commita pelo nome. Turno
  `done` segue para uma sessão revisora — modo `plan`, na worktree do filho, só com `read`, `glob`,
  `grep`, `symbol` e `verdict` —, e revisão sem veredito roda de novo uma vez, depois fica `in_review`
  com relato. `changes_requested` volta ao mesmo filho em duas fases: na `red` ele escreve só o teste
  que falha e o nomeia com a ferramenta nova `red_test`; o harness o roda naquele commit, antes do
  conserto, e só aceita o conserto depois de o teste falhar (`red_test.confirmed`). Depois da
  aprovação, o harness faz o merge da ponta da main — a do `origin`, depois de um `fetch`, com forge
  `github` — na branch, no lugar de integração — conflito volta
  ao filho com os marcadores, e o commit seguinte do harness conclui o merge —, e mede o gate ele mesmo
  no resultado, com a ponta da main como `base` e a alegação do filho ao lado. Gate vermelho volta em
  fase `fix`; verde é empurrado, ganha PR e uma pergunta à pessoa (medido de novo depois de `gate_stale`,
  mantém o PR, empurrado sem `--force` e com a evidência refeita), e só o `merge` dela, pelo novo
  `POST /v1/sessions/{id}/questions/{qid}`, põe o filho na fila serial: ele entra na main, a worktree
  sai, e fica `merged`; `return` devolve, `close` fecha. A orquestradora ganha `ask` para perguntas de
  desenho, e `GET /v1/sessions/{id}/questions` as lista. Além de `orchestrator.max_returns` (3) o filho
  vai a `failed`, `incomplete`. `review.verdict`, `gate.measured`, `question.*`, `merge.*` e
  `main.health` chegam ao fio da orquestradora. Todo desfecho, e toda resposta, entra na caixa da
  orquestradora — em disco, com as perguntas de merge e a fila, entregue depois de um reinício —, e os
  relatos que chegam dentro de `orchestrator.report_debounce` (2s) viram um turno dela com origem
  `children`, nunca no meio de um turno e acrescentado ao fim, sem mudar o contexto que o modelo já
  leu. Passos 7 a 10 da primeira fatia do orquestrador, ligados. Antes eram componentes sem chamador:
  o filho que terminava ficava `working`, e nada o revisava, media ou fazia o merge.
