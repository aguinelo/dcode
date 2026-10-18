# Um merge espera a pessoa

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.p` §4, §7.4–§7.6, §13, §14; `.config` §3; `.i`, passos 9 e 10).

## O que mudou

O núcleo do passo 10, em `internal/orchestrate/`:

- `integrate.go`: o **lugar de integração**, `<orchestrator.dir>/<id>/integration`, uma worktree
  destacada do repositório, criada uma vez e recusada se cair dentro da árvore da pessoa.
  `Integration.Update` atualiza a branch do filho com `git merge --no-ff` da main sobre o `Head`, e a
  branch, na worktree do filho, avança para o `H` resultante — só para a frente, nada reescrito.
  Conflito é abortado na integração, que termina limpa, refeito na worktree do filho com os
  marcadores, e volta como `*Conflict` com os caminhos (`child.returned`); a orquestradora não
  resolve conteúdo (Q-H).
- `forge.go` e `forge_local.go`: `MergeFrom` lê `orchestrator.forge`, `ci_timeout`, `sensitive` e
  `sensitive_diff_lines`, que entram em `KnownKeys`; `NewForge` resolve `auto` em `github` (com
  `origin` e um `gh auth status` que responde) ou `local`, e `github` declarado sem `gh` é erro,
  nunca queda para `local`. No GitHub: push sem forçar, `gh pr create`, `gh pr merge <n> --squash
  --match-head-commit <medido>`, a branch remota apagada depois, e a saúde da main lida em `gh run
  list --commit`. Local: squash na integração e `git update-ref refs/heads/<main> S M`, recusado com
  a main em checkout em qualquer worktree, dizendo onde; a saúde da main é o gate rodado no commit do
  merge.
- `merge.go`: `Merges`, as perguntas e a fila de um projeto. `Propose` só aceita gate verde no
  `Head`, abre o PR com um bloco fixo de evidência (`PRBody`) e pergunta (`Question{kind: merge}`,
  com `Sensitive`). `Answer`: a primeira resposta leva; `merge` põe o filho na fila (`merge.queued`);
  `return` e `close` são `merge.refused{person}` com a nota. `Merge` é serial: lê as pontas, lê a
  saúde da main quando a ponta mudou, recusa `main_red` (salvo `fixes_main`), confere `CheckFresh`
  (`gate_stale` tira o filho da fila para medir de novo), faz o merge e lê `main.health` até verde,
  vermelho ou `ci_timeout`.

## O que o código decide onde a `.p` deixava a forma em aberto

- **A `Forge` ganha `Tips`.** A §7.6 pede `fetch` e as duas pontas antes do merge; quem sabe onde
  elas estão é o forge (`origin/<x>` no GitHub, `refs/heads/<x>` no local). A `.p` §4 foi emendada.
- **`merge.queued` é o sim da pessoa**, não a entrada na esteira: a §7.6 diz que a resposta `merge`
  põe o filho na `Queue`. O que a esteira escreve antes de propor é `gate.measured`.
- **A recusa da pessoa é `merge.refused` com o código `person`**, ao lado de `main_red`,
  `gate_stale`, `gate_undeclared` e da recusa do forge (`forge`): o log diz por que um filho medido
  verde não entrou, e o porquê é a nota da pessoa.
- **Main que a fila ainda não leu é lida antes do merge.** Sem isso o primeiro merge de um projeto
  seria sempre `main_red` (desconhecida). `pending` também recusa: só `green` libera.
- **`main_red` deixa o filho na fila**; `gate_stale` e a recusa do forge tiram, porque o próximo
  passo é medir ou perguntar de novo.
- **O corpo do PR nunca leva a saída do gate.** O que um gate imprime pode ter um segredo; a saída
  inteira fica em `gates/`, nesta máquina. O bloco diz commit, main, comando, resultado e veredito.
- **Local não faz push.** A tarefa falava em empurrar a main no merge local; a `.p` §7.5 e §7.6 dizem
  "sem push, sem PR" e a troca condicional da ref, e a `.p` vale. `forge.local` diz, na primeira vez,
  o que se perde: o PR para ler, o CI do forge e a trava no servidor.
- **A branch remota que não sai depois do merge é dita no `why` de `merge.completed`**, e o merge
  fica: ele já aconteceu no servidor.

## Invariantes e o que fica

Sobem para a §13 dez linhas com teste: sem gate verde nada é proposto e o corpo não leva a saída;
o que andou sai da fila; nada faz merge antes do sim; `--match-head-commit`; a troca condicional;
a recusa com a main em checkout; a fila parada pela main vermelha, salvo `fixes_main`; `pending`
que vira `unknown`; o conflito que volta ao filho; a integração fora da árvore da pessoa.

A fila de perguntas e a de merge no `State` (persistidas, com `forge.local` dito uma vez por
projeto), as rotas de `/questions`, a ferramenta `ask`, e a chamada na esteira — o `Update` antes
do gate, o `Propose` depois dele, o `Merge` no sim — ficam para a fiação dos passos 7 e 10. Até lá
`MergeFrom`, `NewForge`, `NewMerges`, `OpenIntegration` e `Propose` ficam dispensados em
`internal/specguard/unused_orchestrate_test.go`; `CheckFresh` e sete nomes do fio ganharam uso e
saíram de lá.
