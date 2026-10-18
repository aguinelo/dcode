## en

- **A child's report starts the orchestrator's turn, and the pipeline runs on its own.** When a
  child's turn ends, the harness first checks `git status` against the child's owns: a change outside
  them is committed by nobody, reviewed by nobody, and handed back (`outside_owns`). Inside, the harness
  commits by name. If the turn ended done, a reviewer session follows — plan mode, in the child's
  worktree, with only `read`, `glob`, `grep`, `symbol` and `verdict` — and a review that ends without a
  verdict runs once more, then stays `in_review` and is reported. `changes_requested` goes back to the
  same child in two phases. In the red phase the child writes only the failing test and names it with
  the new `red_test` tool; the harness runs it on that commit before any fix and takes the fix only once
  it fails (`red_test.confirmed`). After an approval the harness measures the gate itself, on the
  child's exact commit with main's tip as its base, and keeps the child's own claim beside the
  measurement. A red gate goes back in the fix phase. A green one is handed to the merge step. Past
  `orchestrator.max_returns` (3) the child fails `incomplete`. `review.verdict` and `gate.measured`
  reach the orchestrator's wire. Every outcome, and the person's answer to a question, goes into the
  orchestrator's inbox, which is kept on disk and delivered after a restart. Reports that arrive within
  `orchestrator.report_debounce` (2s) start one orchestrator turn with origin `children`, never in the
  middle of a turn, and appended so the model's context before it is unchanged. Steps 7 to 9 of the
  orchestrator's first slice, wired. Before, those steps were components with no caller: a child that
  finished stayed `working`, and nothing reviewed or measured it.

## pt-BR

- **O relato de um filho começa o turno da orquestradora, e a esteira corre sozinha.** Ao fim de um
  turno de filho, o harness confere o `git status` contra o `owns` antes de tudo: mudança fora dele não
  é commitada nem revisada, e volta ao filho (`outside_owns`); dentro, o harness commita pelo nome. Turno
  `done` segue para uma sessão revisora — modo `plan`, na worktree do filho, só com `read`, `glob`,
  `grep`, `symbol` e `verdict` —, e revisão sem veredito roda de novo uma vez, depois fica `in_review`
  com relato. `changes_requested` volta ao mesmo filho em duas fases: na `red` ele escreve só o teste
  que falha e o nomeia com a ferramenta nova `red_test`; o harness o roda naquele commit, antes do
  conserto, e só aceita o conserto depois de o teste falhar (`red_test.confirmed`). Depois da
  aprovação, o harness mede o gate ele mesmo, no commit exato do filho, com a ponta da main como
  `base`, e a alegação do filho ao lado da medição; gate vermelho volta em fase `fix`, verde vai ao
  passo de merge. Além de `orchestrator.max_returns` (3) o filho vai a `failed`, `incomplete`.
  `review.verdict` e `gate.measured` chegam ao fio da orquestradora. Todo desfecho, e a resposta da
  pessoa a uma pergunta, entra na caixa da orquestradora — em disco, entregue depois de um reinício —,
  e os relatos que chegam dentro de `orchestrator.report_debounce` (2s) viram um turno dela com origem
  `children`, nunca no meio de um turno e acrescentado ao fim, sem mudar o contexto que o modelo já
  leu. Passos 7 a 9 da primeira fatia do orquestrador, ligados. Antes eram componentes sem chamador: o
  filho que terminava ficava `working`, e nada o revisava nem o media.
