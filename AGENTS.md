# dcode

An agentic coding harness written in Go: a daemon, a terminal client, and the
agent loop between them, shipped as one static binary. No cgo outside the
isolated package.

These are the instructions for **any** agent working in this repository —
`dcode` itself reads this file, and so does Claude Code through `CLAUDE.md`.
Keep it about the repository. Tool-specific configuration belongs in the file
of the tool that has it.

## Rules

- Do what has been asked; nothing more, nothing less
- NEVER create files unless absolutely necessary — prefer editing existing files
- NEVER create documentation files unless explicitly requested
- NEVER save working files or tests to the root
- ALWAYS read a file before editing it
- NEVER commit secrets, credentials, or `.env` files
- NEVER add a `Co-Authored-By` trailer to a commit. It is authorship attribution
  under git and GitHub convention; a tool is the facilitator, not a co-author.
- Keep files under 500 lines
- Validate input at system boundaries
- Every failure fails explicitly, with a visible warning — an LLM failure, a
  timeout, a dead end the code cannot get past, or an internal error. Never
  silent, never a state that looks like success, never a bare fallback with
  nothing said about why. See `docs/conventions/GO-CODE-REVIEW.md`'s Errors
  section for what this looks like at the code level.

## Working agreement — branches, staging, CI

Every rule below exists because it was broken, and each names the failure so it
can be recognised before it repeats rather than after.

### One theme, one branch, one PR

- A theme is what fits in a PR title **without the word "and"**. If the title
  needs an "and", it is two branches.
- Before starting work whose title would not fit the open PR, cut a new branch
  from an updated `main`. Continuing on the open branch because it is already
  checked out is how a PR reaches 11 commits and 17k lines — unreviewable, and
  merged on trust rather than on reading.
- A defect found while doing something else gets **its own branch**, unless
  fixing it is what unblocks the current work. "It was right there" is not a
  reason to widen a PR.
- Repeated approval to continue (`segue`, `pode ir`) grants **the work**, never
  the branch. Ask which branch, or start a new one.

### Toda alteração entra no `CHANGELOG.md` — como fragmento

- Na **mesma branch** que a faz, como **um arquivo novo**:
  `changelog.d/<yyyymmddhhmm>-<slug>.md`, com `## en` e `## pt-BR`, cada um com
  uma entrada só. Changelog escrito depois é changelog reconstruído de memória.
- **Nenhum PR edita linha compartilhada.** Em 16/10/2026 quinze PRs em voo
  conflitaram todos nas mesmas linhas — o topo do `Unreleased` nas duas edições
  e a linha contada dos changelogs de decisão — e a integração virou serial.
  O `Unreleased` / `Não publicado` commitado é só um ponteiro: as entradas saem
  dos fragmentos (`make changelog` mostra), e o release
  (`scripts/changelog-assemble.sh release`) as dobra na seção da versão e apaga
  os fragmentos. O "Estado atual" é reescrito no release, não em cada PR.
- Uma entrada por mudança: **o que mudou e por quê**. O detalhe da decisão
  continua no changelog da família, em `docs/specs/architecture/<família>/changelog/`.
- **Os números do "Estado atual" são contados, não digitados — pelo release.**
  `TestTheStateTableIsCountedAndNotCarried` lê famílias, changelogs de decisão,
  contratos declarados, quantos precisam de modelo, quantos se resolvem por
  asserção, quantos já foram de fato medidos e quantos não sabem nomear o
  prompt. **Antes** conferia a tabela contra a árvore em todo commit, e por isso
  todo PR que criava um changelog de família editava a mesma linha. **Agora**:
  sem fragmento pendente — o estado que o release deixa, antes de o commit dele
  passar na CI e virar tag — a tabela tem de ser igual à árvore, linha a linha,
  como antes; e sempre, o escritor que o release roda reescreve a tabela a
  partir da árvore e ela tem de ler de volta como a árvore, a prosa tem de
  bater com a própria tabela, e as duas edições uma com a outra. Entre releases,
  tabela atrás da árvore é dita no log do teste (`make changelog` mostra), nunca
  calada; número digitado à mão é sobrescrito pelo release e reprova se o
  release for pulado.
- `TestEveryFragmentCarriesBothEditions`, `TestNoUnreleasedEntryIsWrittenByHand`
  e `TestTheAssemblerRendersExactlyTheFragments` conferem que todo fragmento
  tem nome datado e as duas edições não vazias, uma entrada cada; que nenhuma
  edição tem entrada escrita à mão no `Unreleased`; e que o montador entrega
  exatamente os fragmentos.
- **O que a guarda ainda não cobre é a metade em prosa da regra**: que o
  fragmento exista, que esteja na branch que fez a mudança, que diga o porquê,
  e que as duas edições digam a mesma coisa — a guarda vê que as duas existem,
  não que se traduzem.
  Isso continua valendo o que vale a disciplina de quem lê, e está escrito aqui
  em vez de combinado. Cobertura e versão publicada também ficam de fora, de
  propósito: a primeira sai de uma execução que o teste não faz, e a segunda é
  afirmação sobre uma tag que um teste sem rede não enxerga.
- **O que muda em `desktop/` entra no changelog do desktop** — um fragmento em
  `desktop/changelog.d/`, que monta o `desktop/CHANGELOG.md` e a edição pt-BR —,
  não neste. O app desktop tem
  versão e tags próprias (`desktop-v*`). Uma mudança que atravessa os dois, como
  um pedido de protocolo que o desktop consome, entra nos dois.

### A versão é derivada, não decidida

- `./scripts/version.sh` diz qual é a próxima, a partir dos commits desde a
  última tag. `./scripts/changelog-assemble.sh release <versão>` dobra os
  fragmentos na seção dela; `./scripts/changelog.sh` lista os PRs que entraram,
  para conferir que nenhum ficou sem fragmento — **o porquê de cada entrada é
  escrito à mão**, no fragmento, porque não sai de gerador nenhum.
- Os dois leem só o núcleo: commit que mexe apenas em `desktop/` não conta, e as
  tags `desktop-v*` não servem de ponto de partida.
- Comportamento faz parte do contrato: contrato removido, limiar que desce, ou
  descrição de ferramenta que muda de sentido é **no mínimo MINOR**. O SemVer lê
  assinaturas, e a superfície deste produto é em parte feita de frases.
- O detalhe está em `docs/conventions/VERSIONING.md`.

### Stage explicitly

- **Never `git add -A` or `git add .`.** Stage named paths. A 41KB HTML mock
  entered a PR unnoticed exactly this way.
- Run `git status` before committing and account for **every** listed file. A
  file you cannot explain is a file that does not belong in the commit.

### The gate runs here; CI confirms on main

- **The local gate is what a pull request is merged on.** `.githooks/pre-push`
  runs the gate CI runs on the commit being pushed, in a throwaway worktree —
  never the working tree, so an ignored or unstaged file cannot make it pass —
  and refuses the push if it fails. Enable it once per clone with
  `make hooks`; it covers every worktree. **Never push with `--no-verify`.** A
  red gate is fixed, not skipped. The gate takes minutes: give the push a long
  enough timeout.
- Pull requests do not run CI: minutes spent finding on GitHub what a local run
  finds in seconds are minutes wasted. A pull request can still run it on
  demand (`gh workflow run ci --ref <branch>`), and should when it touches what
  only Linux exercises — the bubblewrap sandbox.
- A change to `desktop/` also runs the rulers the hook does not:
  `npm run check:visual` and `npm run check:daemon`, when it touches what they
  measure. Say in the PR that they ran.
- Never merge a pull request whose push did not go through the hook.
- **CI runs on `main`, after the merge, and its result is read.** A red `main`
  is the next thing fixed, before anything else merges.

### What the build depends on must be in the repository

- After creating a file the build, the gate or CI depends on, verify it is
  actually tracked: `git check-ignore -v <path>` and `git ls-files <path>`.
- `.gitignore` patterns without a leading `/` match at **any depth**.
  `coverage.*` swallowed `scripts/coverage.sh`, so the coverage gate never ran
  in CI while every local run reported it green.

## Build and test

```bash
make check   # lint, eval compile, race, coverage gate, build — the whole gate
make test    # the suite alone
make cover   # the suite with the coverage gate
make install # build and install into ~/.local/bin
```

`make check` is the Go half of the gate the pre-push hook runs; the hook runs it
on the pushed commit, which is the claim that counts.

Behavioural contracts live behind the `eval` build tag and are **not run** by
`make check`: each scenario runs against a real model and costs money. Run them
with `make eval`. `make check` and CI do compile them (`make eval-build`), on
every change: code behind a build tag that nothing compiles rots in silence, and
it rots in the pull requests that move the code under it, not in the ones that
touch it.

## Where the rules are written

These four are the source of truth. Do not restate them here; a fifth copy is a
copy that drifts.

| Document | What it governs |
|---|---|
| `docs/conventions/SDD-HARNESS.md` | the spec protocol: `.r` → `.p` → `.config` / `.i`, and `changelog/` |
| `docs/conventions/TESTING.md` | TDD, reproducing tests, the coverage gate, critical scenarios |
| `docs/conventions/GO-CODE-REVIEW.md` | what Go code in this repository looks like |
| `docs/conventions/LANGUAGE.md` | which language each artefact is written in |

The short version of the last one, because it is the one most easily broken:
**specs and changelogs in Portuguese; code, comments, commit messages and pull
request titles in English.**

`docs/ROADMAP.md` is what is known and not yet done. `docs/DECISIONS.md` records
why the specs say what they say where the reason came from the code.
