## en

- **The daemon reads and opens a branch's PR through `gh`.** `GET /v1/git/pr`
  answers the PR of the branch a workspace has checked out now — number,
  address, state, draft, and its checks summed up as `pass`, `fail`, `pending`
  or `none` and counted — and the base a new one would target. `POST
  /v1/git/pr` pushes the branch with `git push -u origin <branch>`, never by
  force, and only then runs `gh pr create` with the title, body, base and draft
  a person confirmed; nothing merges and nothing turns auto-merge on. Every
  refusal comes before the push and says why: `gh_unavailable` for gh missing
  or logged out, `pr_refused` for a detached head, no origin, the base itself,
  nothing to propose, an unknown base, a PR already open or a push the remote
  refused. Through gh because it already holds the person's login, so the
  daemon never touches a token; in the daemon so the desktop's changes panel
  and the TUI ask the same reader. The client gains `ReadGitPR` and
  `CreateGitPR`.

## pt-BR

- **O daemon lê e abre o PR de uma branch pelo `gh`.** `GET /v1/git/pr`
  responde o PR da branch que um workspace tem agora — número, endereço,
  estado, rascunho, e as checagens somadas em `pass`, `fail`, `pending` ou
  `none` e contadas — e a base que um PR novo teria. `POST /v1/git/pr` empurra
  a branch com `git push -u origin <branch>`, nunca com força, e só depois roda
  `gh pr create` com o título, o corpo, a base e o rascunho que uma pessoa
  confirmou; nada faz merge e nada liga auto-merge. Toda recusa vem antes do
  push e diz por quê: `gh_unavailable` para `gh` ausente ou sem login,
  `pr_refused` para HEAD destacado, sem `origin`, a própria base, nada a
  propor, base desconhecida, PR já aberto ou push que o remoto recusou. Pelo
  `gh` porque ele já guarda o login da pessoa, e o daemon nunca toca num token;
  no daemon para o painel de alterações do desktop e a TUI perguntarem ao mesmo
  leitor. O cliente ganha `ReadGitPR` e `CreateGitPR`.
