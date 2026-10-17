## en

- **Changes are recorded as fragments.** Each change is one new file,
  `changelog.d/<yyyymmddhhmm>-<slug>.md`, carrying the entry under `## en` and
  `## pt-BR`; `Unreleased` is assembled from them (`make changelog` prints it)
  and `scripts/changelog-assemble.sh release` folds them into the version's
  section and deletes them. On 16 October fifteen pull requests in flight all
  conflicted on the top of `Unreleased` in both editions and on the counted
  decision-changelog row, and integration went serial. The state-table guard
  changes with it: it held the table to the tree on every commit, which is
  what made every pull request edit that row; it now holds it exactly when no
  fragment is pending — the state the release leaves — runs the release's
  writer on every change, and logs the drift between releases. The 40
  entries of `Unreleased` moved into fragments unchanged.

## pt-BR

- **Mudanças são registradas como fragmentos.** Cada mudança é um arquivo
  novo, `changelog.d/<yyyymmddhhmm>-<slug>.md`, com a entrada em `## en` e
  `## pt-BR`; o `Não publicado` é montado a partir deles (`make changelog` o
  imprime) e o `scripts/changelog-assemble.sh release` os dobra na seção da
  versão e os apaga. Em 16 de outubro quinze PRs em voo conflitaram todos no
  topo do `Não publicado` nas duas edições e na linha contada dos changelogs de
  decisão, e a integração virou serial. A guarda da tabela de estado muda
  junto: ela exigia a tabela igual à árvore em todo commit, e é isso que fazia
  todo PR editar aquela linha; agora exige exatamente quando não há fragmento
  pendente — o estado que o release deixa —, roda o escritor do release em
  toda mudança e registra no log a distância entre releases. As 40 entradas do
  `Não publicado` foram para fragmentos sem mudança.
