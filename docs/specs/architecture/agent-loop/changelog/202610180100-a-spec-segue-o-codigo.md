# A spec segue o código

**Data:** 2026-10-18
**Specs afetadas:** `202608072335-agent-loop` (`.p`)

> **Regra:** correção de documentação. Nenhum comportamento muda; a spec passa
> a dizer o que `internal/loop` e `internal/protocol` já fazem.

## O que estava errado

- **Números de antes.** A seção 3 dizia que loop patológico aparece "em três
  repetições" — o default é cinco desde `202609151600`, e o `.config` já dizia
  cinco. O comentário de `Limits` dizia `MaxIterations // default 50`, quando
  zero pede o teto da família: `claude` 50, `minimax-m3` 2000. A justificativa
  dos 50 fica, agora como a de uma família, e a prosa sobre "cinquenta
  iterações são o teto real" passa a falar do teto da família.
- **Um diagrama de fases.** A seção 2 abria dizendo que não há segunda máquina
  de estados por fase (`202608110945`) e fechava com um diagrama de transições
  `assembling → streaming → executing → blocked`. O diagrama saiu; no lugar,
  uma frase diz que a sequência é o laço de `Engine.Run`.
- **Onde os motivos de parada vivem.** A spec declarava `type StopReason
  string` em `internal/loop`. Os motivos são constantes string sem tipo em
  `internal/protocol`, carregadas em `TurnCompleted.Reason`. Faltava
  `StopDaemonRestart`, que a reabertura de sessão emite e o loop não — por isso
  ele fica de fora da guarda de motivos alcançáveis pelo loop. `StopRefused`
  ganhou o porquê do nome próprio.
- **Dirigir o turno.** `loop.Config.Steer` e o evento `turn.steered` existiam
  sem uma linha no `.p`. A seção 6.1 descreve o contrato — anexado como a
  pessoa, nunca como lembrete, drenado em ordem no topo de cada rodada — e
  aponta a decisão em `client-tui/202608170400`.
- **O comentário de `IsRepeat`** em `internal/loop/limits.go` dizia "three
  times"; diz agora que dispara em `n` repetições, cinco por default. É a única
  linha de código desta correção, e é comentário.
- **O índice de changelog** não listava `202608110900`, `202608110930` nem
  `202608110945`.

## O que não mudou

Nenhum limite, nenhum motivo, nenhuma invariante. As linhas da seção 8 ficam
como estavam, e com elas as reivindicações em `internal/loop/invariants_test.go`.
