# Worktree meio feita sai

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.p` §7.4, §8, invariantes).

## O que aconteceu

O CI da main ficou vermelho no macOS em `TestAGateTheCallerStopsIsAnErrorAndNotAResult`. Quem chama
cancelou enquanto o `git worktree add` da worktree descartável do gate ainda rodava; o git foi morto
no meio e deixou a worktree **trancada** (`lock reason: initializing`). A limpeza rodava
`git worktree remove --force` uma vez, e um `--force` só recusa worktree trancada: a worktree ficava
para trás, e o erro devolvido era o `signal: killed` do git, não o cancelamento.

## O que mudou

- `forceRemove` (`worktree.go`) tira a worktree em qualquer estado: `git worktree remove -f -f`, o
  diretório removido se o git não o removeu, e `git worktree prune`. Erro só quando algo **fica** — no
  disco ou na lista do git —, e a mensagem diz qual; passo que falhou e foi coberto pelo seguinte não
  é resíduo.
- A limpeza roda num contexto próprio (`dropHalfMade`), nunca no de quem chamou, que pode ser o motivo
  da falha.
- `removeTree` do gate e `removeWorktree` do filho passam por ela. `addWorktree` e `OpenIntegration`
  removem o que um `add` falho deixou — só um caminho que não existia antes do `add`. Lugar de
  integração meio feito era recusado em toda abertura seguinte como "não é a worktree de integração".
- Gate parado por quem chamou, durante o `rev-parse` ou o `worktree add`, devolve o cancelamento
  (`errors.Is(err, context.Canceled)`), com o que o git disse ao lado; falha da limpeza vai junto, não
  no lugar.
- O `pre-push` usa `-f -f`, remove o diretório e roda `prune` no `trap`.

## Por que um teste sem relógio

O teste do CI dependia de o cancelamento cair, aos 200 ms, depois do `add`. Os novos não dependem de
tempo: um faz a worktree, a tranca com `--reason initializing` e chama a limpeza; o outro troca o
`add` por uma costura (`addGateTree`) que deixa a worktree trancada, cancela e devolve o erro do git
morto.
