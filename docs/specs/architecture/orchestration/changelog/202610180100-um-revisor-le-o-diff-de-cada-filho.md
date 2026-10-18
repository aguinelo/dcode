# Um revisor lê o diff de cada filho

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §4, §7.2, §13, §14; `.config` §2, §3; `.i`, passo 8).

## O que mudou

O núcleo do passo 8, construído antes da esteira que o chama: `internal/orchestrate/review.go`,
`internal/orchestrate/review_verdict.go` e a ferramenta `internal/tools/verdict.go`.

- `AssembleReview` monta o que o revisor lê a partir da worktree do filho: a tarefa, o `owns`, os
  arquivos de `Base..Head`, o diff e as specs que a tarefa nomeia. Arquivo do diff fora do `owns` é
  um `*OwnsViolation`, com o `Return{outside_owns, paths}`, e nenhuma revisão: nenhum modelo é
  consultado sobre um diff que o filho não podia fazer.
- `Uncommitted` lê o `git status --porcelain` da worktree e `OutsideOwns` confere cada caminho
  contra o `owns` (caminho exato, diretório acima, ou `policy.Glob`): é a conferência da §7.1, antes
  de qualquer commit, e pega o que o filho escreveu pelo `bash`.
- `ReviewerRegistry` é o registro do revisor: `read`, `glob`, `grep`, `symbol` e `verdict`, montado
  do zero e não filtrado do registro de uma sessão, para que uma ferramenta nova não chegue ao
  revisor sem alguém pô-la ali.
- A ferramenta `verdict{verdict, findings, why}` termina a revisão. `Review.Submit` recusa o que não
  fecha: `approve` com achados, `changes_requested` sem achado, achado sem caminho, `what` ou `why`,
  veredito sem `why`, `missing` dito pelo modelo, e um segundo veredito.
- `Review.Finish` escreve `review.verdict` no log de decisões, com o veredito em `measured` e o
  `why` do revisor. Revisão que termina sem `verdict` grava `missing`, com o porquê do harness, e
  devolve `ErrNoVerdict`: falha explícita, nunca aprovação.
- `ReviewFrom` lê `orchestrator.reviewer_profiles` e `orchestrator.review_max_diff_bytes`, que
  entram em `KnownKeys`; `ReviewConfig.Profile` escolhe o perfil dentro da lista, e fora dela é
  `model_not_allowed`.

## O que o código decide onde a `.p` deixava a forma em aberto

- **Diff acima do limite sai inteiro, não cortado.** A `.p` dizia "acima disso, a lista de arquivos
  e a instrução de lê-los"; o código não manda nem um pedaço do diff, porque meio diff lido como o
  todo é o defeito que o limite existe para não criar. O texto ao revisor diz "Diff cut", o tamanho,
  e os arquivos a ler.
- **`outside_owns` é do harness, não do veredito.** O revisor diz `approve` ou `changes_requested`;
  o `owns` é conferido antes, sobre o diff, sem modelo. Um modelo não decide se o filho podia
  escrever um caminho.
- **`verdict` leva `why`.** A §9.1 pede o porquê do modelo quando ele escolhe; a ferramenta o
  exige, e é ele que vai ao log.
- **As specs que a tarefa nomeia** são lidas do texto da tarefa (`*.spec.md`): o caminho como
  escrito, ou o único arquivo com aquele nome quando a tarefa o nomeia solto. Nome que o commit não
  tem, ou que mais de um arquivo tem, vai em `MissingSpecs` e é dito ao revisor, nunca descartado.
- **`reviewer_profiles` vazio cai no primeiro de `child_profiles`**, que é chave do passo 6: o
  `Profile` recebe essa queda de quem chama, para que este passo não leia uma chave de outro.
- **`ReviewRequest`** ganha `Base`, `Task`, `Owns`, `Specs`, `MissingSpecs`, `DiffBytes` e
  `DiffCut`; `Prompt` é o primeiro turno do revisor, em inglês (`LANGUAGE.md`).

## Invariantes e o que fica

Sobem para a §13: o revisor sem ferramenta que escreve (da §14), e duas novas — revisão sem
`verdict` é `missing` e nunca aprovação; diff acima do limite sai inteiro e é dito. Fica na §14 o
"volta ao filho sem commit e sem revisão", cuja metade "sem commit" é da esteira.

Ficam para a fiação: `Spawner.SpawnReviewer` (a sessão `RoleReviewer` em modo `plan` na worktree do
filho, com este registro e `Prompt` como primeiro turno), o commit do harness com os caminhos
nomeados, `owns.checked` no log, o `missing` que roda de novo uma vez, a devolução em duas fases com
`red_test`, `orchestrator.max_returns` e o estado `in_review`. Até lá `AssembleReview`, `Uncommitted`,
`ReviewFrom`, `ReviewerRegistry`, `NewReview` e `Finish` não têm chamador fora dos testes e ficam
dispensados em `internal/specguard/unused_orchestrate_test.go`, como os do gate.
